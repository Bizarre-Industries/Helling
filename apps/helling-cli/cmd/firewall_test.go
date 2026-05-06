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

func runFirewall(t *testing.T, args []string) (string, error) {
	t.Helper()
	return runFirewallWithInput(t, args, "")
}

func runFirewallWithInput(t *testing.T, args []string, stdin string) (string, error) {
	t.Helper()
	root := cmd.NewFirewallCmd()
	root.PersistentFlags().String("api", "", "")
	root.PersistentFlags().String("output", "", "")
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs(args)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(stdin))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := root.ExecuteContext(ctx)
	return buf.String(), err
}

func TestFirewallList_PrintsTable(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"fw1","direction":"input","action":"drop","protocol":"tcp","source_cidr":"203.0.113.0/24","destination_port":22,"nft_comment":"helling:fw1"}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runFirewall(t, []string{"list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "fw1") || !strings.Contains(out, "helling:fw1") {
		t.Fatalf("out: %q", out)
	}
}

func TestFirewallAdd_PostsStructuredRule(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/firewall/host" {
			t.Fatalf("request: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"fw1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	if _, err := runFirewall(t, []string{
		"add",
		"--direction=output",
		"--action=reject",
		"--protocol=udp",
		"--source-cidr=10.0.0.0/24",
		"--destination-cidr=198.51.100.0/24",
		"--destination-port=53",
		"--comment=dns-egress",
	}); err != nil {
		t.Fatal(err)
	}
	if gotBody["direction"] != "output" || gotBody["action"] != "reject" || gotBody["protocol"] != "udp" {
		t.Fatalf("body: %+v", gotBody)
	}
	if gotBody["source_cidr"] != "10.0.0.0/24" || gotBody["destination_cidr"] != "198.51.100.0/24" || gotBody["comment"] != "dns-egress" {
		t.Fatalf("body: %+v", gotBody)
	}
	if gotBody["destination_port"] != float64(53) {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestFirewallDelete_CallsDelete(t *testing.T) {
	useTempConfigDir(t)
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runFirewall(t, []string{"delete", "fw1", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/firewall/host/fw1" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "deleted fw1") {
		t.Fatalf("out: %q", out)
	}
}

func TestFirewallDelete_RequiresConfirmation(t *testing.T) {
	useTempConfigDir(t)
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runFirewallWithInput(t, []string{"delete", "fw1"}, "no\n")
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if called {
		t.Fatal("DELETE should not be called without confirmation")
	}
	if !strings.Contains(out, "Type yes to confirm") {
		t.Fatalf("out: %q", out)
	}
}
