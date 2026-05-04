package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsInvalidSocketConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "relative socket path",
			config: `
server:
  socket_path: run/helling/api.sock
`,
			wantErr: "server.socket_path must be absolute",
		},
		{
			name: "world writable socket mode",
			config: `
server:
  socket_mode: 511
`,
			wantErr: "server.socket_mode must grant no broader access than 0660",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tt.config))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadRejectsInvalidAuthConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name: "zero login rate limit",
			config: `
auth:
  login_rate_limit_per_15m: 0
`,
			wantErr: "auth.login_rate_limit_per_15m must be > 0",
		},
		{
			name: "empty setup token path",
			config: `
auth:
  setup_token_path: ""
`,
			wantErr: "auth.setup_token_path must not be empty",
		},
		{
			name: "relative setup token path",
			config: `
auth:
  setup_token_path: setup-token
`,
			wantErr: "auth.setup_token_path must be absolute",
		},
		{
			name: "zero argon2 time cost",
			config: `
auth:
  argon2_time_cost: 0
`,
			wantErr: "auth.argon2_time_cost must be > 0",
		},
		{
			name: "excessive argon2 time cost",
			config: `
auth:
  argon2_time_cost: 11
`,
			wantErr: "auth.argon2_time_cost must be <= 10",
		},
		{
			name: "excessive argon2 memory",
			config: `
auth:
  argon2_memory_kib: 262145
`,
			wantErr: "auth.argon2_memory_kib must be <= 262144",
		},
		{
			name: "zero argon2 parallelism",
			config: `
auth:
  argon2_parallelism: 0
`,
			wantErr: "auth.argon2_parallelism must be > 0",
		},
		{
			name: "excessive argon2 parallelism",
			config: `
auth:
  argon2_parallelism: 9
`,
			wantErr: "auth.argon2_parallelism must be <= 8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tt.config))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadAcceptsDefaultConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load(defaults): %v", err)
	}
	if got := cfg.Server.SocketMode; got != 0o660 {
		t.Fatalf("Server.SocketMode = %o, want 0660", got)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "helling.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
