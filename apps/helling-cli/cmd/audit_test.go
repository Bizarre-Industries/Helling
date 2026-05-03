package cmd_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

func runAudit(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewAuditCmd()
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

func TestAuditQuery_ForwardsFilters(t *testing.T) {
	useTempConfigDir(t)
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"data":[{"id":"aud_1","timestamp":"2026-04-23T10:00:00Z","actor":"admin","action":"auth.login","outcome":"allow"}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runAudit(t, []string{"query", "--actor=admin", "--action=auth.login", "--limit=10"})
	if err != nil {
		t.Fatalf("query: %v out=%q", err, out)
	}
	if !strings.Contains(gotURL, "actor=admin") || !strings.Contains(gotURL, "action=auth.login") {
		t.Fatalf("url: %s", gotURL)
	}
	if !strings.Contains(out, "aud_1") {
		t.Fatalf("out: %q", out)
	}
}

func TestAuditExport_JSONDefault(t *testing.T) {
	useTempConfigDir(t)
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"data":{"format":"json","body":"[...]"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAudit(t, []string{"export"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "format=json") {
		t.Fatalf("url: %s", gotURL)
	}
}

func TestAuditExport_CSVFlag(t *testing.T) {
	useTempConfigDir(t)
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"data":{"format":"csv","body":"id,actor\n..."}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runAudit(t, []string{"export", "--format=csv"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "format=csv") {
		t.Fatalf("url: %s", gotURL)
	}
}
