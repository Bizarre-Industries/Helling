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

func runUser(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewUserCmd()
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

func TestUserList_TableFormat(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"u1","username":"alice","role":"admin","status":"active"}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runUser(t, []string{"list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "alice") || !strings.Contains(out, "admin") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestUserCreate_PostsBody(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"u1","username":"bob","role":"user","status":"active"},"meta":{"request_id":"r"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runUser(t, []string{"create", "bob", "--role=user", "--password=fixture-pw"})
	if err != nil {
		t.Fatalf("create: %v out=%q", err, out)
	}
	if gotBody["username"] != "bob" || gotBody["role"] != "user" || gotBody["password"] != "fixture-pw" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestUserSetScope_PutsBody(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/scope") {
			t.Fatalf("path: %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"scope":"incus:admin"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runUser(t, []string{"set-scope", "u1", "incus:admin"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["scope"] != "incus:admin" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestUserDelete_CallsDelete(t *testing.T) {
	useTempConfigDir(t)
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method: %s", r.Method)
		}
		called = true
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runUser(t, []string{"delete", "u1"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("DELETE not called")
	}
}

func TestUserGet_PrintsRaw(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":"u1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runUser(t, []string{"get", "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "u1") {
		t.Fatalf("out: %q", out)
	}
}

func TestUser_RequiresLogin(t *testing.T) {
	useTempConfigDir(t)
	if _, err := runUser(t, []string{"list"}); err == nil {
		t.Fatal("expected error when not logged in")
	}
}
