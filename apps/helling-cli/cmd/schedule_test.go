package cmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/cmd"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

func runSchedule(t *testing.T, args []string) (string, error) {
	t.Helper()
	return runScheduleWithInput(t, args, "")
}

func runScheduleWithInput(t *testing.T, args []string, stdin string) (string, error) {
	t.Helper()
	root := cmd.NewScheduleCmd()
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

func TestScheduleList_PrintsTable(t *testing.T) {
	useTempConfigDir(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"sch1","name":"nightly","kind":"backup","target":"vm-web-1","on_calendar":"daily","enabled":true}]}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runSchedule(t, []string{"list"})
	if err != nil {
		t.Fatalf("list: %v out=%q", err, out)
	}
	if !strings.Contains(out, "nightly") || !strings.Contains(out, "ON_CALENDAR") {
		t.Fatalf("out: %q", out)
	}
}

func TestScheduleCreate_PostsBody(t *testing.T) {
	useTempConfigDir(t)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/schedules" {
			t.Fatalf("request: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"sch1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	if _, err := runSchedule(t, []string{"create", "nightly", "--kind=snapshot", "--target=vm-db-1", "--on-calendar=hourly"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "nightly" || gotBody["kind"] != "snapshot" || gotBody["target"] != "vm-db-1" || gotBody["on_calendar"] != "hourly" {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestScheduleGet_PrintsRaw(t *testing.T) {
	useTempConfigDir(t)
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"id":"sch1","name":"nightly"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runSchedule(t, []string{"get", "sch1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/v1/schedules/sch1" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "nightly") {
		t.Fatalf("out: %q", out)
	}
}

func TestScheduleUpdate_PatchesChangedFields(t *testing.T) {
	useTempConfigDir(t)
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"data":{"id":"sch1","enabled":false}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	if _, err := runSchedule(t, []string{"update", "sch1", "--name=weekly", "--on-calendar=weekly", "--enabled=false"}); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/api/v1/schedules/sch1" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if gotBody["name"] != "weekly" || gotBody["on_calendar"] != "weekly" || gotBody["enabled"] != false {
		t.Fatalf("body: %+v", gotBody)
	}
}

func TestScheduleRun_PostsRunEndpoint(t *testing.T) {
	useTempConfigDir(t)
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"data":{"id":"evt1"}}`))
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runSchedule(t, []string{"run", "sch1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/schedules/sch1/run" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "evt1") {
		t.Fatalf("out: %q", out)
	}
}

func TestScheduleRunSystemUsesRunnerToken(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), "schedule-runner.token")
	if err := os.WriteFile(tokenPath, []byte("runner-token\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	t.Setenv("HELLING_SCHEDULE_TOKEN_FILE", tokenPath)

	var gotMethod, gotPath, gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotToken = r.Header.Get("X-Helling-Schedule-Token")
		_, _ = w.Write([]byte(`{"data":{"id":"evt-system"}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("HELLING_API", srv.URL)

	out, err := runSchedule(t, []string{"run", "sch1", "--system"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/internal/schedules/sch1/run" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if gotToken != "runner-token" {
		t.Fatalf("schedule token header: got %q", gotToken)
	}
	if !strings.Contains(out, "evt-system") {
		t.Fatalf("out: %q", out)
	}
}

func TestScheduleDelete_CallsDelete(t *testing.T) {
	useTempConfigDir(t)
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runSchedule(t, []string{"delete", "sch1", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/schedules/sch1" {
		t.Fatalf("request: %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(out, "deleted sch1") {
		t.Fatalf("out: %q", out)
	}
}

func TestScheduleDelete_RequiresConfirmation(t *testing.T) {
	useTempConfigDir(t)
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	t.Cleanup(srv.Close)
	seedProfile(t, config.Profile{API: srv.URL, AccessToken: "jwt.x"})

	out, err := runScheduleWithInput(t, []string{"delete", "sch1"}, "no\n")
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
