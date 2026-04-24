// Package config loads and persists the helling CLI's on-disk profile.
//
// Location follows XDG per docs/spec/cli.md: $XDG_CONFIG_HOME/helling/config.yaml
// or $HOME/.config/helling/config.yaml when XDG_CONFIG_HOME is unset.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile is the per-user persisted state.
type Profile struct {
	// API is the hellingd endpoint base URL, e.g. http://127.0.0.1:8080.
	API string `yaml:"api"`
	// RefreshCookie is the helling_refresh cookie captured during login. Opaque
	// to the CLI; surfaced only in the Cookie header for /api/v1/auth/refresh.
	RefreshCookie string `yaml:"refresh_cookie,omitempty"`
	// AccessToken is the last-issued JWT access token. Short-lived (15m) so
	// the CLI auto-refreshes via RefreshCookie when verification fails.
	AccessToken string `yaml:"access_token,omitempty"`
	// Token is an operator-supplied API token (helling_*) for non-interactive
	// automation. When set, overrides AccessToken.
	Token string `yaml:"token,omitempty"`
}

// Path returns the default on-disk config path.
func Path() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "helling", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home: %w", err)
	}
	return filepath.Join(home, ".config", "helling", "config.yaml"), nil
}

// Load reads the config file. Returns an empty Profile + nil when the file
// does not exist (fresh install). Explicit path overrides the default.
func Load(explicit string) (Profile, error) {
	path := explicit
	if path == "" {
		p, err := Path()
		if err != nil {
			return Profile{}, err
		}
		path = p
	}
	raw, err := os.ReadFile(path) //nolint:gosec // path is operator-controlled
	if errors.Is(err, os.ErrNotExist) {
		return Profile{}, nil
	}
	if err != nil {
		return Profile{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	var prof Profile
	if err := yaml.Unmarshal(raw, &prof); err != nil {
		return Profile{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return prof, nil
}

// Save writes the profile to disk with 0600 permissions, creating parent
// directories as needed. Explicit path overrides the default.
func Save(prof *Profile, explicit string) error {
	path := explicit
	if path == "" {
		p, err := Path()
		if err != nil {
			return err
		}
		path = p
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}
	raw, err := yaml.Marshal(prof)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// Bearer returns the preferred Bearer token: operator Token overrides
// AccessToken. Empty when neither is set.
func (p *Profile) Bearer() string {
	if p.Token != "" {
		return p.Token
	}
	return p.AccessToken
}
