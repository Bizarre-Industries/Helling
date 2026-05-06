// Package systemd renders and links Helling-managed systemd units.
package systemd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	// DefaultStagingDir is the group-writable schedule unit staging directory.
	DefaultStagingDir = "/etc/systemd/system/helling-managed"
	// DefaultHelperPath is the setuid helper used for schedule unit links.
	DefaultHelperPath = "/usr/lib/helling/helling-unit-link"
	// DefaultHellingCLIPath is the packaged CLI path used by generated units.
	DefaultHellingCLIPath = "/usr/bin/helling"
	scheduleUnitPrefix    = "helling-schedule-"
	helperInstallAction   = "install"
	helperRemoveAction    = "remove"
)

var managedScheduleUnitName = regexp.MustCompile(`^helling-schedule-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\.(service|timer)$`)

// ScheduleSpec contains the public schedule fields needed to render units.
type ScheduleSpec struct {
	ID         string
	Kind       string
	Target     string
	OnCalendar string
	Enabled    bool
}

// ScheduleUnits holds the paired service and timer unit contents.
type ScheduleUnits struct {
	ServiceName string
	TimerName   string
	Service     []byte
	Timer       []byte
}

// ScheduleManager writes rendered unit files and invokes the privileged helper.
type ScheduleManager struct {
	StagingDir     string
	HelperPath     string
	HellingCLIPath string
	RunHelper      func(path string, args ...string) error
}

// ValidateScheduleInput checks public schedule fields before persistence.
func ValidateScheduleInput(kind, target, onCalendar string) error {
	return validateScheduleSpec(ScheduleSpec{
		ID:         "00000000-0000-0000-0000-000000000000",
		Kind:       kind,
		Target:     target,
		OnCalendar: onCalendar,
	})
}

// RenderScheduleUnits returns the service and timer units for a schedule.
func RenderScheduleUnits(spec ScheduleSpec, hellingCLIPath string) (ScheduleUnits, error) {
	if err := validateScheduleSpec(spec); err != nil {
		return ScheduleUnits{}, err
	}
	if hellingCLIPath == "" {
		hellingCLIPath = DefaultHellingCLIPath
	}
	if !filepath.IsAbs(hellingCLIPath) || containsControlOrSpace(hellingCLIPath) {
		return ScheduleUnits{}, fmt.Errorf("helling CLI path %q must be an absolute path without whitespace or control characters", hellingCLIPath)
	}

	serviceName := scheduleUnitPrefix + spec.ID + ".service"
	timerName := scheduleUnitPrefix + spec.ID + ".timer"
	if err := ValidateManagedUnitName(serviceName); err != nil {
		return ScheduleUnits{}, err
	}
	if err := ValidateManagedUnitName(timerName); err != nil {
		return ScheduleUnits{}, err
	}

	var service bytes.Buffer
	service.WriteString("[Unit]\n")
	service.WriteString("Description=Helling schedule ")
	service.WriteString(spec.ID)
	service.WriteByte('\n')
	service.WriteString("Documentation=https://github.com/Bizarre-Industries/Helling\n\n")
	service.WriteString("[Service]\n")
	service.WriteString("Type=oneshot\n")
	service.WriteString("User=helling\n")
	service.WriteString("Group=helling\n")
	service.WriteString("Environment=HELLING_API=http+unix:///run/helling/api.sock\n")
	service.WriteString("Environment=HELLING_SCHEDULE_TOKEN_FILE=/etc/helling/schedule-runner.token\n")
	service.WriteString("ExecStart=")
	service.WriteString(hellingCLIPath)
	service.WriteString(" schedule run --system ")
	service.WriteString(spec.ID)
	service.WriteString("\n")

	var timer bytes.Buffer
	timer.WriteString("[Unit]\n")
	timer.WriteString("Description=Helling schedule timer ")
	timer.WriteString(spec.ID)
	timer.WriteByte('\n')
	timer.WriteString("Documentation=https://github.com/Bizarre-Industries/Helling\n\n")
	timer.WriteString("[Timer]\n")
	timer.WriteString("OnCalendar=")
	timer.WriteString(spec.OnCalendar)
	timer.WriteByte('\n')
	timer.WriteString("Persistent=true\n")
	timer.WriteString("Unit=")
	timer.WriteString(serviceName)
	timer.WriteString("\n\n")
	timer.WriteString("[Install]\n")
	timer.WriteString("WantedBy=timers.target\n")

	return ScheduleUnits{
		ServiceName: serviceName,
		TimerName:   timerName,
		Service:     service.Bytes(),
		Timer:       timer.Bytes(),
	}, nil
}

// ValidateManagedUnitName ensures the helper only receives known schedule unit basenames.
func ValidateManagedUnitName(name string) error {
	if name == "" {
		return errors.New("unit name is required")
	}
	if filepath.Base(name) != name {
		return fmt.Errorf("unit name %q must be a basename", name)
	}
	if !managedScheduleUnitName.MatchString(name) {
		return fmt.Errorf("unit name %q is not a managed schedule service or timer", name)
	}
	return nil
}

// Install writes schedule units and asks the helper to install service then timer.
func (m ScheduleManager) Install(ctx context.Context, spec ScheduleSpec) error {
	units, err := RenderScheduleUnits(spec, m.hellingCLIPath())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.stagingDir(), 0o770); err != nil { //nolint:gosec // directory must be root:helling 0770 per ADR-017.
		return fmt.Errorf("creating schedule unit staging dir: %w", err)
	}
	if err := writeUnit(filepath.Join(m.stagingDir(), units.ServiceName), units.Service); err != nil {
		return err
	}
	if err := writeUnit(filepath.Join(m.stagingDir(), units.TimerName), units.Timer); err != nil {
		return err
	}
	if err := m.runHelper(ctx, helperInstallAction, units.ServiceName); err != nil {
		return fmt.Errorf("installing schedule service unit: %w", err)
	}
	if spec.Enabled {
		if err := m.runHelper(ctx, helperInstallAction, units.TimerName); err != nil {
			return fmt.Errorf("installing schedule timer unit: %w", err)
		}
	}
	return nil
}

// Remove asks the helper to disable and remove both units. Missing units are handled by the helper.
func (m ScheduleManager) Remove(ctx context.Context, id string) error {
	spec := ScheduleSpec{ID: id, Kind: "backup", Target: "unused", OnCalendar: "daily"}
	units, err := RenderScheduleUnits(spec, m.hellingCLIPath())
	if err != nil {
		return err
	}
	if err := m.runHelper(ctx, helperRemoveAction, units.TimerName); err != nil {
		return fmt.Errorf("removing schedule timer unit: %w", err)
	}
	if err := m.runHelper(ctx, helperRemoveAction, units.ServiceName); err != nil {
		return fmt.Errorf("removing schedule service unit: %w", err)
	}
	return nil
}

func validateScheduleSpec(spec ScheduleSpec) error {
	switch {
	case spec.ID == "":
		return errors.New("schedule id is required")
	case spec.Kind != "backup" && spec.Kind != "snapshot":
		return errors.New("schedule kind must be backup or snapshot")
	case strings.TrimSpace(spec.Target) == "":
		return errors.New("schedule target is required")
	case strings.TrimSpace(spec.OnCalendar) == "":
		return errors.New("schedule on_calendar is required")
	case containsControl(spec.OnCalendar):
		return errors.New("schedule on_calendar must not contain control characters")
	case containsControl(spec.Target):
		return errors.New("schedule target must not contain control characters")
	}
	return nil
}

func (m ScheduleManager) stagingDir() string {
	if m.StagingDir != "" {
		return m.StagingDir
	}
	return DefaultStagingDir
}

func (m ScheduleManager) helperPath() string {
	if m.HelperPath != "" {
		return m.HelperPath
	}
	return DefaultHelperPath
}

func (m ScheduleManager) hellingCLIPath() string {
	if m.HellingCLIPath != "" {
		return m.HellingCLIPath
	}
	return DefaultHellingCLIPath
}

func (m ScheduleManager) runHelper(ctx context.Context, action, unit string) error {
	if action != helperInstallAction && action != helperRemoveAction {
		return fmt.Errorf("unsupported helper action %q", action)
	}
	if err := ValidateManagedUnitName(unit); err != nil {
		return err
	}
	if m.RunHelper != nil {
		return m.RunHelper(m.helperPath(), action, unit)
	}
	// #nosec G204 -- helper path is fixed by default or operator-configured; action/unit are validated above.
	cmd := exec.CommandContext(ctx, m.helperPath(), action, unit)
	return cmd.Run()
}

func writeUnit(path string, body []byte) error {
	if err := ValidateManagedUnitName(filepath.Base(path)); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".helling-unit-*")
	if err != nil {
		return fmt.Errorf("creating temporary unit file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temporary unit file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temporary unit file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod temporary unit file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("installing staged unit file: %w", err)
	}
	return nil
}

func containsControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func containsControlOrSpace(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
