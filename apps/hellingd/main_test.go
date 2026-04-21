package main

import (
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestMainUsesInjectedExitAndServe(t *testing.T) {
	origServe := defaultServe
	origExit := exitFunc
	origStderr := stderr
	t.Cleanup(func() {
		defaultServe = origServe
		exitFunc = origExit
		stderr = origStderr
	})

	defaultServe = func(*http.Server) error {
		return http.ErrServerClosed
	}
	stderr = io.Discard

	var code int
	exitFunc = func(c int) {
		code = c
	}

	main()

	if code != 0 {
		t.Fatalf("expected exit code 0 from main, got %d", code)
	}
}

func TestRunReturnsZeroOnServerClosed(t *testing.T) {
	logger := newLogger(io.Discard)
	code := run(logger, func(*http.Server) error {
		return http.ErrServerClosed
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestRunReturnsOneOnServeFailure(t *testing.T) {
	logger := newLogger(io.Discard)
	code := run(logger, func(*http.Server) error {
		return errors.New("boom")
	})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestRunConfiguresHTTPServer(t *testing.T) {
	logger := newLogger(io.Discard)
	var got *http.Server

	code := run(logger, func(server *http.Server) error {
		got = server
		return http.ErrServerClosed
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if got == nil {
		t.Fatal("expected server to be initialized")
	}

	if got.Addr != defaultAddr {
		t.Fatalf("expected addr %q, got %q", defaultAddr, got.Addr)
	}

	if got.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("expected ReadHeaderTimeout %v, got %v", 10*time.Second, got.ReadHeaderTimeout)
	}
}
