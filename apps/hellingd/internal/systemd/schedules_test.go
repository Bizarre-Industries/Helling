package systemd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const testScheduleID = "018f3a40-0000-7000-8000-000000000000"

func TestRenderScheduleUnits(t *testing.T) {
	t.Parallel()
	spec := ScheduleSpec{
		ID:         testScheduleID,
		Kind:       "backup",
		Target:     "vm-web-1",
		OnCalendar: "*-*-* 02:00:00 UTC",
	}

	units, err := RenderScheduleUnits(spec, "/usr/bin/helling")
	if err != nil {
		t.Fatalf("RenderScheduleUnits: %v", err)
	}

	timer := string(units.Timer)
	for _, want := range []string{
		"[Timer]\n",
		"OnCalendar=*-*-* 02:00:00 UTC\n",
		"Persistent=true\n",
		"Unit=helling-schedule-" + testScheduleID + ".service\n",
	} {
		if !bytes.Contains(units.Timer, []byte(want)) {
			t.Fatalf("timer missing %q:\n%s", want, timer)
		}
	}

	service := string(units.Service)
	for _, want := range []string{
		"[Service]\n",
		"Type=oneshot\n",
		"Environment=HELLING_API=http+unix:///run/helling/api.sock\n",
		"Environment=HELLING_SCHEDULE_TOKEN_FILE=/etc/helling/schedule-runner.token\n",
		"ExecStart=/usr/bin/helling schedule run --system " + testScheduleID + "\n",
	} {
		if !bytes.Contains(units.Service, []byte(want)) {
			t.Fatalf("service missing %q:\n%s", want, service)
		}
	}
}

func TestRenderScheduleUnitsDefaultsToPackagedCLIPath(t *testing.T) {
	t.Parallel()
	spec := ScheduleSpec{
		ID:         testScheduleID,
		Kind:       "backup",
		Target:     "vm-web-1",
		OnCalendar: "daily",
	}

	units, err := RenderScheduleUnits(spec, "")
	if err != nil {
		t.Fatalf("RenderScheduleUnits: %v", err)
	}

	want := "ExecStart=/usr/bin/helling schedule run --system " + testScheduleID + "\n"
	if !bytes.Contains(units.Service, []byte(want)) {
		t.Fatalf("service missing default CLI path %q:\n%s", want, string(units.Service))
	}
}

func TestRenderScheduleUnitsRejectsUnitInjection(t *testing.T) {
	t.Parallel()
	_, err := RenderScheduleUnits(ScheduleSpec{
		ID:         testScheduleID,
		Kind:       "backup",
		Target:     "vm-web-1",
		OnCalendar: "daily\nExecStart=/bin/sh",
	}, "/usr/bin/helling")
	if err == nil {
		t.Fatal("RenderScheduleUnits accepted newline injection")
	}
}

func TestValidateManagedUnitName(t *testing.T) {
	t.Parallel()
	valid := []string{
		"helling-schedule-" + testScheduleID + ".service",
		"helling-schedule-" + testScheduleID + ".timer",
	}
	for _, name := range valid {
		if err := ValidateManagedUnitName(name); err != nil {
			t.Fatalf("ValidateManagedUnitName(%q): %v", name, err)
		}
	}

	invalid := []string{
		"helling-schedule-" + testScheduleID + ".socket",
		"helling-schedule-../evil.service",
		"/etc/systemd/system/helling-schedule-" + testScheduleID + ".service",
		"other-" + testScheduleID + ".timer",
		"helling-schedule-not-a-uuid.timer",
	}
	for _, name := range invalid {
		if err := ValidateManagedUnitName(name); err == nil {
			t.Fatalf("ValidateManagedUnitName(%q) unexpectedly succeeded", name)
		}
	}
}

func TestScheduleManagerInstallWritesUnitsAndInvokesHelper(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	var calls [][]string
	manager := ScheduleManager{
		StagingDir:     stageDir,
		HelperPath:     "/usr/lib/helling/helling-unit-link",
		HellingCLIPath: "/usr/bin/helling",
		RunHelper: func(_ string, args ...string) error {
			calls = append(calls, append([]string(nil), args...))
			return nil
		},
	}

	err := manager.Install(t.Context(), ScheduleSpec{
		ID:         testScheduleID,
		Kind:       "snapshot",
		Target:     "vm-web-1",
		OnCalendar: "daily",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, suffix := range []string{".service", ".timer"} {
		path := filepath.Join(stageDir, "helling-schedule-"+testScheduleID+suffix)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode: got %o want 600", path, info.Mode().Perm())
		}
	}
	wantCalls := [][]string{
		{"install", "helling-schedule-" + testScheduleID + ".service"},
		{"install", "helling-schedule-" + testScheduleID + ".timer"},
	}
	if len(calls) != len(wantCalls) {
		t.Fatalf("helper calls: got %#v want %#v", calls, wantCalls)
	}
	for i := range wantCalls {
		if calls[i][0] != wantCalls[i][0] || calls[i][1] != wantCalls[i][1] {
			t.Fatalf("helper call %d: got %#v want %#v", i, calls[i], wantCalls[i])
		}
	}
}
