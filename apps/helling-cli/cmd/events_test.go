package cmd_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

func runEvents(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewEventsCmd()
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

func TestEventsTail_Default(t *testing.T) {
	useTempConfigDir(t)
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"data":[{"id":"evt_1","type":"instance.created","timestamp":"2026-04-23T10:05:00Z","source":"incus"}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runEvents(t, []string{"tail"})
	if err != nil {
		t.Fatalf("tail: %v out=%q", err, out)
	}
	if gotURL != "/api/v1/events" {
		t.Fatalf("url: %s", gotURL)
	}
	if !strings.Contains(out, "evt_1") {
		t.Fatalf("out: %q", out)
	}
}

func TestEventsTail_LimitFlag(t *testing.T) {
	useTempConfigDir(t)
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runEvents(t, []string{"tail", "--limit=25"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotURL, "limit=25") {
		t.Fatalf("url: %s", gotURL)
	}
}
