package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSocketListenerSetsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.sock")
	listener, err := newSocketListener(path, "", 0o660)
	if err != nil {
		t.Fatalf("newSocketListener: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o660 {
		t.Fatalf("socket mode = %v, want 0660", got)
	}
}
