package systemd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitLinkerInstallTimerLinksAndEnables(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	systemDir := t.TempDir()
	unit := "helling-schedule-" + testScheduleID + ".timer"
	if err := os.WriteFile(filepath.Join(stageDir, unit), []byte("[Timer]\nOnCalendar=daily\n"), 0o640); err != nil {
		t.Fatalf("write staged timer: %v", err)
	}
	var commands []commandCall
	linker := UnitLinker{
		StagingDir: stageDir,
		SystemDir:  systemDir,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			commands = append(commands, commandCall{name: name, args: append([]string(nil), args...)})
			return nil
		},
	}

	if err := linker.Handle(t.Context(), []string{"install", unit}); err != nil {
		t.Fatalf("Handle install: %v", err)
	}

	linkTarget, err := os.Readlink(filepath.Join(systemDir, unit))
	if err == nil {
		t.Fatalf("installed unit is a symlink to %q, want canonical regular file", linkTarget)
	}
	body, err := os.ReadFile(filepath.Join(systemDir, unit))
	if err != nil {
		t.Fatalf("read installed unit: %v", err)
	}
	if !strings.Contains(string(body), "OnCalendar=daily") {
		t.Fatalf("installed timer body missing OnCalendar: %s", string(body))
	}
	assertCommandCalls(t, commands, []commandCall{
		{name: systemctlPath, args: []string{"daemon-reload"}},
		{name: systemctlPath, args: []string{"enable", "--now", unit}},
	})
}

func TestUnitLinkerRemoveTimerDisablesAndUnlinks(t *testing.T) {
	t.Parallel()
	stageDir := t.TempDir()
	systemDir := t.TempDir()
	unit := "helling-schedule-" + testScheduleID + ".timer"
	staged := filepath.Join(stageDir, unit)
	if err := os.WriteFile(staged, []byte("[Timer]\nOnCalendar=daily\n"), 0o640); err != nil {
		t.Fatalf("write staged timer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(systemDir, unit), []byte("[Timer]\nOnCalendar=daily\n"), 0o644); err != nil {
		t.Fatalf("write installed timer: %v", err)
	}
	var commands []commandCall
	linker := UnitLinker{
		StagingDir: stageDir,
		SystemDir:  systemDir,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			commands = append(commands, commandCall{name: name, args: append([]string(nil), args...)})
			return nil
		},
	}

	if err := linker.Handle(t.Context(), []string{"remove", unit}); err != nil {
		t.Fatalf("Handle remove: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(systemDir, unit)); !os.IsNotExist(err) {
		t.Fatalf("installed unit exists after remove: %v", err)
	}
	assertCommandCalls(t, commands, []commandCall{
		{name: systemctlPath, args: []string{"disable", "--now", unit}},
		{name: systemctlPath, args: []string{"daemon-reload"}},
	})
}

func TestUnitLinkerRejectsUnsafeArgs(t *testing.T) {
	t.Parallel()
	linker := UnitLinker{StagingDir: t.TempDir(), SystemDir: t.TempDir()}
	for _, args := range [][]string{
		{"install"},
		{"install", "../helling-schedule-" + testScheduleID + ".timer"},
		{"remove", "ssh.service"},
		{"restart", "helling-schedule-" + testScheduleID + ".timer"},
	} {
		if err := linker.Handle(t.Context(), args); err == nil {
			t.Fatalf("Handle(%v) unexpectedly succeeded", args)
		}
	}
}

type commandCall struct {
	name string
	args []string
}

func assertCommandCalls(t *testing.T, got, want []commandCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("commands: got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i].name != want[i].name {
			t.Fatalf("command %d name: got %q want %q", i, got[i].name, want[i].name)
		}
		if len(got[i].args) != len(want[i].args) {
			t.Fatalf("command %d args: got %#v want %#v", i, got[i].args, want[i].args)
		}
		for j := range want[i].args {
			if got[i].args[j] != want[i].args[j] {
				t.Fatalf("command %d arg %d: got %q want %q", i, j, got[i].args[j], want[i].args[j])
			}
		}
	}
}
