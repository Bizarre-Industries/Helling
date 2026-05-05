package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/systemd"
)

func TestCreateScheduleInstallsSystemdUnits(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	fakeUnits := &fakeScheduleUnitManager{}
	srv.cfg.ScheduleUnits = fakeUnits
	seedAdminUser(t, st)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/schedules", map[string]any{
		"name":        "nightly",
		"kind":        "backup",
		"target":      "vm-web-1",
		"on_calendar": "daily",
	}, cookie)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create status: got %d want 201 body=%s", resp.StatusCode, string(body))
	}
	if len(fakeUnits.installed) != 1 {
		t.Fatalf("installed units: got %#v want one install", fakeUnits.installed)
	}
	if fakeUnits.installed[0].Kind != "backup" || fakeUnits.installed[0].Target != "vm-web-1" || fakeUnits.installed[0].OnCalendar != "daily" {
		t.Fatalf("installed spec: %#v", fakeUnits.installed[0])
	}
}

func TestCreateScheduleRejectsOnCalendarInjectionBeforeInstall(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	fakeUnits := &fakeScheduleUnitManager{}
	srv.cfg.ScheduleUnits = fakeUnits
	seedAdminUser(t, st)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	resp := postJSON(t, ts.Client(), ts.URL+"/v1/schedules", map[string]any{
		"name":        "nightly",
		"kind":        "backup",
		"target":      "vm-web-1",
		"on_calendar": "daily\nExecStart=/bin/sh",
	}, cookie)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create status: got %d want 400 body=%s", resp.StatusCode, string(body))
	}
	if len(fakeUnits.installed) != 0 {
		t.Fatalf("installed units despite invalid calendar: %#v", fakeUnits.installed)
	}
	rows, err := st.ListSchedules(t.Context(), "")
	if err != nil {
		t.Fatalf("ListSchedules: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("stored schedules: got %d want 0", len(rows))
	}
}

func TestDeleteScheduleRemovesSystemdUnits(t *testing.T) {
	t.Parallel()
	srv, st := newTestServer(t)
	fakeUnits := &fakeScheduleUnitManager{}
	srv.cfg.ScheduleUnits = fakeUnits
	u := seedAdminUser(t, st)
	row, err := st.CreateSchedule(t.Context(), u.ID, "nightly", "backup", "vm-web-1", "daily")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, ts.URL+"/v1/schedules/"+row.ID, http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE schedule: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("delete status: got %d want 204 body=%s", resp.StatusCode, string(body))
	}
	if len(fakeUnits.removed) != 1 || fakeUnits.removed[0] != row.ID {
		t.Fatalf("removed units: got %#v want %q", fakeUnits.removed, row.ID)
	}
}

func TestRunBackupScheduleCreatesIncusBackup(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-backup-op"}
	srv, st := newServerWithIncus(t, fake)
	u := seedAdminUser(t, st)
	row, err := st.CreateSchedule(t.Context(), u.ID, "nightly", "backup", "vm-web-1", "daily")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/schedules/"+row.ID+"/run", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST schedule run: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("run status: got %d want 202 body=%s", resp.StatusCode, string(body))
	}
	if fake.backupTarget != "vm-web-1" || fake.backupName == "" {
		t.Fatalf("backup call: target=%q name=%q", fake.backupTarget, fake.backupName)
	}
}

func TestRunSnapshotScheduleCreatesIncusSnapshot(t *testing.T) {
	t.Parallel()
	fake := &fakeIncusClient{nextOpID: "incus-snapshot-op"}
	srv, st := newServerWithIncus(t, fake)
	u := seedAdminUser(t, st)
	row, err := st.CreateSchedule(t.Context(), u.ID, "hourly", "snapshot", "vm-web-1", "hourly")
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	cookie := loginCookie(t, ts, "admin", testPassword)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+"/v1/schedules/"+row.ID+"/run", http.NoBody)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.AddCookie(cookie)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("POST schedule run: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("run status: got %d want 202 body=%s", resp.StatusCode, string(body))
	}
	if fake.snapshotTarget != "vm-web-1" || fake.snapshotName == "" {
		t.Fatalf("snapshot call: target=%q name=%q", fake.snapshotTarget, fake.snapshotName)
	}
}

func seedAdminUser(t *testing.T, st *store.Store) store.User {
	t.Helper()
	hash, err := auth.Hash(testPassword, auth.Argon2Params{Time: 1, MemoryKiB: 8 * 1024, Parallelism: 1, SaltLen: 16, KeyLen: 32})
	if err != nil {
		t.Fatalf("auth.Hash: %v", err)
	}
	u, err := st.CreateUser(context.Background(), "admin", hash, true)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

type fakeScheduleUnitManager struct {
	installed []systemd.ScheduleSpec
	removed   []string
}

func (f *fakeScheduleUnitManager) Install(_ context.Context, spec systemd.ScheduleSpec) error {
	f.installed = append(f.installed, spec)
	return nil
}

func (f *fakeScheduleUnitManager) Remove(_ context.Context, id string) error {
	f.removed = append(f.removed, id)
	return nil
}
