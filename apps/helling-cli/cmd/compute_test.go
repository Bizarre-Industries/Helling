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

func runCompute(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewComputeCmd()
	root.PersistentFlags().String("api", "", "")
	root.PersistentFlags().String("output", "", "")
	root.SetArgs(args)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := root.ExecuteContext(ctx)
	return buf.String(), err
}

func TestComputeList_IncusEnvelopeFormatsTable(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/incus/1.0/instances") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"metadata":[{"name":"vm-web","status":"Running","type":"virtual-machine"},{"name":"ct-db","status":"Stopped","type":"container"}]}`))
	}))
	t.Cleanup(srv.Close)

	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runCompute(t, []string{"list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "vm-web") || !strings.Contains(out, "ct-db") {
		t.Fatalf("unexpected table: %q", out)
	}
}

func TestComputeList_EmptyResponse(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":[]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runCompute(t, []string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No instances found") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestComputeList_RequiresLogin(t *testing.T) {
	useTempConfigDir(t)
	_, err := runCompute(t, []string{"list"})
	if err == nil {
		t.Fatal("expected error when not logged in")
	}
}
