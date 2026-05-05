package systemd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultSystemDir is the systemd unit directory that receives managed symlinks.
const (
	DefaultSystemDir = "/etc/systemd/system"
	systemctlPath    = "/usr/bin/systemctl"
)

// UnitLinker implements the root-owned helper operation surface.
type UnitLinker struct {
	StagingDir string
	SystemDir  string
	RunCommand func(ctx context.Context, name string, args ...string) error
}

// Handle validates helper args and executes install/remove.
func (l UnitLinker) Handle(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: helling-unit-link install|remove <helling-schedule-uuid.service|timer>")
	}
	action, unit := args[0], args[1]
	if action != "install" && action != "remove" {
		return fmt.Errorf("unsupported action %q", action)
	}
	if err := ValidateManagedUnitName(unit); err != nil {
		return err
	}

	switch action {
	case "install":
		return l.install(ctx, unit)
	case "remove":
		return l.remove(ctx, unit)
	default:
		return fmt.Errorf("unsupported action %q", action)
	}
}

func (l UnitLinker) install(ctx context.Context, unit string) error {
	body, err := l.canonicalUnitBody(unit)
	if err != nil {
		return err
	}
	target := filepath.Join(l.systemDir(), unit)
	if err := l.prepareTarget(target, filepath.Join(l.stagingDir(), unit)); err != nil {
		return err
	}
	if err := writeInstalledUnit(target, body); err != nil {
		return err
	}
	if err := l.run(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if strings.HasSuffix(unit, ".timer") {
		if err := l.run(ctx, "enable", "--now", unit); err != nil {
			return fmt.Errorf("systemctl enable timer: %w", err)
		}
	}
	return nil
}

func (l UnitLinker) remove(ctx context.Context, unit string) error {
	if strings.HasSuffix(unit, ".timer") {
		if err := l.run(ctx, "disable", "--now", unit); err != nil {
			return fmt.Errorf("systemctl disable timer: %w", err)
		}
	}
	target := filepath.Join(l.systemDir(), unit)
	if err := l.removeTarget(target, filepath.Join(l.stagingDir(), unit)); err != nil {
		return err
	}
	if err := l.run(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	return nil
}

func (l UnitLinker) prepareTarget(target, source string) error {
	return removeManagedTarget(target, source, "replace")
}

func (l UnitLinker) removeTarget(target, source string) error {
	return removeManagedTarget(target, source, "remove")
}

func removeManagedTarget(target, source, action string) error {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat installed unit: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to %s non-regular unit %s", action, target)
		}
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("removing unit: %w", err)
		}
		return nil
	}
	current, err := os.Readlink(target)
	if err != nil {
		return fmt.Errorf("read installed symlink: %w", err)
	}
	if current != source {
		return fmt.Errorf("refusing to %s unit symlink outside staging dir", action)
	}
	if err := os.Remove(target); err != nil {
		return fmt.Errorf("removing unit symlink: %w", err)
	}
	return nil
}

func (l UnitLinker) stagingDir() string {
	if l.StagingDir != "" {
		return l.StagingDir
	}
	return DefaultStagingDir
}

func (l UnitLinker) systemDir() string {
	if l.SystemDir != "" {
		return l.SystemDir
	}
	return DefaultSystemDir
}

func (l UnitLinker) run(ctx context.Context, args ...string) error {
	if l.RunCommand != nil {
		return l.RunCommand(ctx, systemctlPath, args...)
	}
	cmd := exec.CommandContext(ctx, systemctlPath, args...)
	cmd.Env = sanitizedRootHelperEnv()
	return cmd.Run()
}

func (l UnitLinker) canonicalUnitBody(unit string) ([]byte, error) {
	id := strings.TrimPrefix(unit, scheduleUnitPrefix)
	id = strings.TrimSuffix(strings.TrimSuffix(id, ".service"), ".timer")
	onCalendar := "daily"
	if strings.HasSuffix(unit, ".timer") {
		var err error
		onCalendar, err = readStagedOnCalendar(filepath.Join(l.stagingDir(), unit))
		if err != nil {
			return nil, err
		}
	}
	units, err := RenderScheduleUnits(ScheduleSpec{
		ID:         id,
		Kind:       "backup",
		Target:     "managed",
		OnCalendar: onCalendar,
		Enabled:    true,
	}, DefaultHellingCLIPath)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(unit, ".timer") {
		return units.Timer, nil
	}
	return units.Service, nil
}

func readStagedOnCalendar(path string) (string, error) {
	body, err := os.ReadFile(path) // #nosec G304 -- path is a validated managed basename under the staging dir.
	if err != nil {
		return "", fmt.Errorf("reading staged timer: %w", err)
	}
	if len(body) > 4096 {
		return "", errors.New("staged timer is too large")
	}
	for _, line := range strings.Split(string(body), "\n") {
		value, ok := strings.CutPrefix(line, "OnCalendar=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if err := ValidateScheduleInput("backup", "managed", value); err != nil {
			return "", err
		}
		return value, nil
	}
	return "", errors.New("staged timer missing OnCalendar")
}

func writeInstalledUnit(path string, body []byte) error {
	if err := ValidateManagedUnitName(filepath.Base(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".helling-unit-*")
	if err != nil {
		return fmt.Errorf("creating installed unit temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing installed unit temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing installed unit temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod installed unit temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("installing canonical unit: %w", err)
	}
	return nil
}

func sanitizedRootHelperEnv() []string {
	return []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
}
