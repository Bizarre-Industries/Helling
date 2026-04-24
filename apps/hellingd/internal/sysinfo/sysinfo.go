// Package sysinfo exposes host metadata used by the /api/v1/system endpoints.
//
// The helpers here are best-effort and cross-platform: production Helling
// installs live on Debian (ADR-002) where the Linux paths are authoritative,
// but developers running macOS need the daemon to start without panicking.
// Where a platform-specific probe is unavailable, the helper returns a
// clearly-marked fallback so the dashboard still renders.
package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Info is the SystemInfo payload.
type Info struct {
	Hostname string
	Version  string
	Uptime   string
	Arch     string
	Kernel   string
}

// Hardware is the SystemHardware payload.
type Hardware struct {
	CPU     string
	Cores   int
	RAMGB   int
	DiskGB  int
	Network string
}

// Start is captured at package init so Uptime remains monotonic per-process.
var start = time.Now()

// Collect returns the live system info.
func Collect(version string) Info {
	host, _ := os.Hostname()
	return Info{
		Hostname: fallback(host, "unknown"),
		Version:  fallback(version, "dev"),
		Uptime:   humanDuration(time.Since(start)),
		Arch:     runtime.GOARCH,
		Kernel:   readKernelRelease(),
	}
}

// CollectHardware probes the host for hardware details. Returns the best
// information available; missing values default to conservative fallbacks.
func CollectHardware() Hardware {
	hw := Hardware{
		CPU:     readCPUModel(),
		Cores:   runtime.NumCPU(),
		RAMGB:   readRAMGB(),
		DiskGB:  0,
		Network: readPrimaryNIC(),
	}
	if hw.CPU == "" {
		hw.CPU = "unknown"
	}
	if hw.Network == "" {
		hw.Network = "unknown"
	}
	if hw.Cores < 1 {
		hw.Cores = 1
	}
	if hw.RAMGB < 1 {
		hw.RAMGB = 1
	}
	if hw.DiskGB < 1 {
		hw.DiskGB = 1
	}
	return hw
}

func fallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func humanDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

// procfsRoot is overridable in tests.
var procfsRoot = "/proc"

// sysfsNetRoot is overridable in tests.
var sysfsNetRoot = "/sys/class/net"

func readKernelRelease() string {
	if v, err := os.ReadFile(procfsRoot + "/sys/kernel/osrelease"); err == nil {
		return strings.TrimSpace(string(v))
	}
	return runtime.GOOS
}

func readCPUModel() string {
	f, err := os.Open(procfsRoot + "/cpuinfo")
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "model name") {
			if _, rest, ok := strings.Cut(line, ":"); ok {
				return strings.TrimSpace(rest)
			}
		}
	}
	return ""
}

func readRAMGB() int {
	f, err := os.Open(procfsRoot + "/meminfo")
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.Atoi(fields[1])
				return kb / 1024 / 1024
			}
		}
	}
	return 0
}

func readPrimaryNIC() string {
	entries, err := os.ReadDir(sysfsNetRoot)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.Name() == "lo" {
			continue
		}
		return e.Name()
	}
	return ""
}
