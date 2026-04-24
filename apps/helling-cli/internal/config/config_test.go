package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.yaml")
	prof, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if prof.API != "" {
		t.Errorf("expected empty profile, got %+v", prof)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "cfg.yaml")
	want := config.Profile{
		API:           "http://127.0.0.1:8080",
		RefreshCookie: "helling_refresh=abc",
		AccessToken:   "jwt.example",
	}
	if err := config.Save(&want, path); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.API != want.API || got.RefreshCookie != want.RefreshCookie || got.AccessToken != want.AccessToken {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestBearerPrefersOperatorToken(t *testing.T) {
	p := config.Profile{Token: "helling_opx", AccessToken: "jwt.x"}
	if p.Bearer() != "helling_opx" {
		t.Fatalf("want operator token, got %q", p.Bearer())
	}
}

func TestBearerFallsBackToAccessToken(t *testing.T) {
	p := config.Profile{AccessToken: "jwt.x"}
	if p.Bearer() != "jwt.x" {
		t.Fatalf("want access token, got %q", p.Bearer())
	}
}

func TestBearerEmptyWhenNeitherSet(t *testing.T) {
	p := config.Profile{}
	if p.Bearer() != "" {
		t.Fatalf("want empty, got %q", p.Bearer())
	}
}

func TestPathHonoursXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Fatalf("unexpected path: %s", got)
	}
}

func TestPathFallsBackToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", t.TempDir())
	got, err := config.Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Fatalf("unexpected path: %s", got)
	}
}

// TestLoadExplicitDefault covers the default-path branch in Load.
func TestLoadExplicitDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	prof, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if prof.API != "" {
		t.Errorf("fresh profile should be empty: %+v", prof)
	}
}

// TestSaveExplicitDefault covers the default-path branch in Save.
func TestSaveExplicitDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p := config.Profile{API: "http://127.0.0.1:8080"}
	if err := config.Save(&p, ""); err != nil {
		t.Fatal(err)
	}
}

// TestLoadBadYAMLReturnsError covers the YAML parse failure.
func TestLoadBadYAMLReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}
