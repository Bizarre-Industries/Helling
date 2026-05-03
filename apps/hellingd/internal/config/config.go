// Package config defines the runtime configuration for hellingd.
//
// Configuration sources, in order of precedence:
//
//  1. CLI flags (handled in main)
//  2. Environment variables (HELLING_* prefix)
//  3. YAML config file at /etc/helling/config.yaml (or path passed via -config)
//  4. Defaults
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config is the top-level runtime configuration.
type Config struct {
	StateDir string       `yaml:"state_dir"`
	Server   ServerConfig `yaml:"server"`
	Log      LogConfig    `yaml:"log"`
	Auth     AuthConfig   `yaml:"auth"`
	Incus    IncusConfig  `yaml:"incus"`
}

// ServerConfig configures the HTTP server's Unix socket.
type ServerConfig struct {
	SocketPath  string      `yaml:"socket_path"`
	SocketGroup string      `yaml:"socket_group"`
	SocketMode  os.FileMode `yaml:"socket_mode"`
}

// LogConfig controls log output format and verbosity.
type LogConfig struct {
	Level  string `yaml:"level"`  // debug | info | warn | error
	Format string `yaml:"format"` // json | text
}

// AuthConfig controls session and password parameters.
type AuthConfig struct {
	SessionTTLHours   int `yaml:"session_ttl_hours"`
	LoginRateLimit    int `yaml:"login_rate_limit_per_15m"`
	Argon2TimeCost    int `yaml:"argon2_time_cost"`
	Argon2MemoryKiB   int `yaml:"argon2_memory_kib"`
	Argon2Parallelism int `yaml:"argon2_parallelism"`
}

// IncusConfig points hellingd at the Incus daemon.
type IncusConfig struct {
	SocketPath string `yaml:"socket_path"` // empty = use INCUS_SOCKET env or default
	Project    string `yaml:"project"`     // Incus project, defaults to "default"
}

// Defaults returns a Config populated with safe defaults.
func Defaults() Config {
	return Config{
		StateDir: "/var/lib/helling",
		Server: ServerConfig{
			SocketPath:  "/run/helling/api.sock",
			SocketGroup: "helling-proxy",
			SocketMode:  0o660,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: AuthConfig{
			SessionTTLHours:   24 * 7,
			LoginRateLimit:    5,
			Argon2TimeCost:    3,
			Argon2MemoryKiB:   64 * 1024,
			Argon2Parallelism: 4,
		},
		Incus: IncusConfig{
			SocketPath: "",
			Project:    "default",
		},
	}
}

// Load reads the YAML file at path, applies environment-variable overrides,
// and returns a fully populated Config. A missing file is not an error;
// defaults are used.
func Load(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path) // #nosec G304 -- path is operator-supplied config file, validated below

	switch {
	case err == nil:
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parsing %s: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		// Fine — use defaults.
	default:
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}

	applyEnv(&cfg)

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HELLING_STATE_DIR"); v != "" {
		cfg.StateDir = v
	}
	if v := os.Getenv("HELLING_SOCKET_PATH"); v != "" {
		cfg.Server.SocketPath = v
	}
	if v := os.Getenv("HELLING_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("HELLING_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("HELLING_INCUS_SOCKET"); v != "" {
		cfg.Incus.SocketPath = v
	}
	if v := os.Getenv("HELLING_INCUS_PROJECT"); v != "" {
		cfg.Incus.Project = v
	}
	if v := os.Getenv("HELLING_SESSION_TTL_HOURS"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			cfg.Auth.SessionTTLHours = h
		}
	}
}

func (c *Config) validate() error {
	if c.StateDir == "" {
		return errors.New("state_dir must not be empty")
	}
	if c.Server.SocketPath == "" {
		return errors.New("server.socket_path must not be empty")
	}
	if c.Auth.SessionTTLHours <= 0 {
		return errors.New("auth.session_ttl_hours must be > 0")
	}
	if c.Auth.Argon2MemoryKiB < 8*1024 {
		return errors.New("auth.argon2_memory_kib must be >= 8192 (8 MiB)")
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		return fmt.Errorf("log.format %q invalid (use json or text)", c.Log.Format)
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level %q invalid", c.Log.Level)
	}
	return nil
}
