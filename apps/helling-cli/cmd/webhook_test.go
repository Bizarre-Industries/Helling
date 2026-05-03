package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

// webhookSecretFixture is a synthetic HMAC-secret placeholder used in tests.
// Real webhook secrets are per-instance random; gitleaks pattern-matches
// generic secret-ish strings, so we keep the value behind a named constant
// with an allow-comment to make the intent explicit.
const webhookSecretFixture = "FIXTURE-SECRET-DO-NOT-USE-IN-PROD" // gitleaks:allow

func runWebhook(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewWebhookCmd()
	root.PersistentFlags().String("api", "", "")
	root.PersistentFlags().String("output", "", "")
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs(args)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := root.ExecuteContext(ctx)
	return buf.String(), err
}

func TestWebhookList_PrintsTable(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"wh1","name":"pager","url":"https://x","events":["a"],"enabled":true}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runWebhook(t, []string{"list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "pager") {
		t.Fatalf("out: %q", out)
	}
}

func TestWebhookCreate_PostsBody(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"wh1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runWebhook(t, []string{
		"create", "w1",
		"--url=https://example.test/h",
		"--secret=" + webhookSecretFixture, // gitleaks:allow
		"--events=instance.created,instance.deleted",
	}); err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "w1" || gotBody["url"] != "https://example.test/h" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestWebhookUpdate_SendsEnabledWhenSet(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method: %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"wh1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runWebhook(t, []string{"update", "wh1", "--enabled=false"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := gotBody["enabled"]; !ok || v != false {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestWebhookTest_CallsTestEndpoint(t *testing.T) {
	useTempConfigDir(t)
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"delivered":"ok"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runWebhook(t, []string{"test", "wh1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/test") {
		t.Fatalf("path: %s", path)
	}
	if !strings.Contains(out, "delivered") {
		t.Fatalf("out: %q", out)
	}
}

func TestWebhookDelete_CallsDelete(t *testing.T) {
	useTempConfigDir(t)
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runWebhook(t, []string{"delete", "wh1"}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete {
		t.Fatalf("method: %s", method)
	}
}
