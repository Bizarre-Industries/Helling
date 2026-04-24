package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCollect_DefaultsWhenVersionEmpty(t *testing.T) {
	info := Collect("")
	if info.Version != "dev" {
		t.Fatalf("expected dev, got %q", info.Version)
	}
	if info.Arch != runtime.GOARCH {
		t.Fatalf("arch mismatch: %q", info.Arch)
	}
	if info.Uptime == "" {
		t.Fatal("uptime empty")
	}
}

func TestCollect_VersionPropagates(t *testing.T) {
	info := Collect("1.2.3")
	if info.Version != "1.2.3" {
		t.Fatalf("version = %q", info.Version)
	}
}

func TestCollectHardware_HasSensibleFallbacks(t *testing.T) {
	hw := CollectHardware()
	if hw.Cores < 1 {
		t.Errorf("cores = %d", hw.Cores)
	}
	if hw.CPU == "" {
		t.Errorf("cpu empty")
	}
	if hw.Network == "" {
		t.Errorf("network empty")
	}
	if hw.RAMGB < 1 {
		t.Errorf("ramgb = %d", hw.RAMGB)
	}
	if hw.DiskGB < 1 {
		t.Errorf("diskgb = %d", hw.DiskGB)
	}
}

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{30 * time.Minute, "30m"},
		{3 * time.Hour, "3h 0m"},
		{25 * time.Hour, "1d 1h"},
	}
	for _, c := range cases {
		if got := humanDuration(c.in); got != c.want {
			t.Errorf("humanDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFallback(t *testing.T) {
	if fallback("", "def") != "def" {
		t.Error("fallback failed")
	}
	if fallback("x", "def") != "x" {
		t.Error("fallback overrode non-empty")
	}
}

// TestProcfsOverrides swaps procfsRoot / sysfsNetRoot to a per-test tmpdir
// so the probe functions exercise their parse paths even on macOS dev hosts.
func TestProcfsOverrides(t *testing.T) {
	dir := writeFakeProcfs(t)
	origProc := procfsRoot
	origNet := sysfsNetRoot
	procfsRoot = dir + "/proc"
	sysfsNetRoot = dir + "/sys/class/net"
	t.Cleanup(func() {
		procfsRoot = origProc
		sysfsNetRoot = origNet
	})

	if got := readKernelRelease(); got != "6.1.0-test" {
		t.Errorf("kernel = %q", got)
	}
	if got := readCPUModel(); got != "Fake CPU @ 3.00GHz" {
		t.Errorf("cpu = %q", got)
	}
	if got := readRAMGB(); got != 16 {
		t.Errorf("ram = %d", got)
	}
	if got := readPrimaryNIC(); got != "eth0" {
		t.Errorf("nic = %q", got)
	}
}

func writeFakeProcfs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must := func(rel, contents string) {
		t.Helper()
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	must("proc/sys/kernel/osrelease", "6.1.0-test\n")
	must("proc/cpuinfo", "processor\t: 0\nmodel name\t: Fake CPU @ 3.00GHz\n")
	must("proc/meminfo", "MemTotal:       16777216 kB\n")
	must("sys/class/net/lo/_", "lo\n")
	must("sys/class/net/eth0/_", "eth0\n")
	return dir
}
