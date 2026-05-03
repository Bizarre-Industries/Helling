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

func runSystem(t *testing.T, args []string) (string, error) {
	t.Helper()
	root := cmd.NewSystemCmd()
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

func TestSystemInfo_Hits(t *testing.T) {
	useTempConfigDir(t)
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"version":"0.1.0-alpha"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	out, err := runSystem(t, []string{"info"})
	if err != nil {
		t.Fatalf("info: %v out=%q", err, out)
	}
	if path != "/api/v1/system/info" {
		t.Fatalf("path: %s", path)
	}
	if !strings.Contains(out, "0.1.0-alpha") {
		t.Fatalf("out: %q", out)
	}
}

func TestSystemHardware_Hits(t *testing.T) {
	useTempConfigDir(t)
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"cpu_count":4}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runSystem(t, []string{"hardware"}); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/system/hardware" {
		t.Fatalf("path: %s", path)
	}
}

func TestSystemDiagnostics_Hits(t *testing.T) {
	useTempConfigDir(t)
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"probes":[]}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runSystem(t, []string{"diagnostics"}); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/system/diagnostics" {
		t.Fatalf("path: %s", path)
	}
}

func TestSystemConfigGet_AppendsKey(t *testing.T) {
	useTempConfigDir(t)
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"value":"info"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runSystem(t, []string{"config-get", "log.level"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "/log.level") {
		t.Fatalf("path: %s", path)
	}
}

func TestSystemConfigPut_SendsValue(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method: %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runSystem(t, []string{"config-set", "log.level", "debug"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["value"] != "debug" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestSystemUpgrade_SendsRollbackFlag(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"status":"scheduled"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})
	if _, err := runSystem(t, []string{"upgrade", "--rollback"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := gotBody["rollback"]; !ok || v != true {
		t.Fatalf("body: %+v", gotBody)
	}
}
