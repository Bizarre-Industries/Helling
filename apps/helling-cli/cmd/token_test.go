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

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

func runAuthTop(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewAuthCmd()
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

func TestAuthTokenList_PrintsTable(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"tok_1","name":"ci-bot","scope":"read","prefix":"helling_pat_abc"}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runAuthTop(t, []string{"token", "list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "ci-bot") || !strings.Contains(out, "helling_pat_abc") {
		t.Fatalf("out: %q", out)
	}
}

func TestAuthTokenCreate_ForwardsBody(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"tok_1","plaintext_token":"helling_pat_xxxxxxxx"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAuthTop(t, []string{"token", "create", "ci-bot", "--scope=write", "--expires-in=3600"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "ci-bot" || gotBody["scope"] != "write" {
		t.Fatalf("body: %+v", gotBody)
	}
	if v, ok := gotBody["expires_in_seconds"]; !ok || v.(float64) != 3600 {
		t.Fatalf("expires: %+v", gotBody)
	}
}

func TestAuthTokenRevoke_DeletesByID(t *testing.T) {
	useTempConfigDir(t)
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAuthTop(t, []string{"token", "revoke", "tok_1"}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/api/v1/auth/tokens/tok_1" {
		t.Fatalf("method=%s path=%s", method, path)
	}
}
