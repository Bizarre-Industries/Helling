package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunWritesOpenAPIDocument(t *testing.T) {
	var buf bytes.Buffer
	if err := run(&buf); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "openapi: 3.1.0") {
		t.Fatalf("expected openapi header in output, got: %s", out)
	}

	if !strings.Contains(out, "/api/v1/health:") {
		t.Fatalf("expected health path in output, got: %s", out)
	}

	if !strings.Contains(out, "operationId: healthGet") {
		t.Fatalf("expected health operation id in output, got: %s", out)
	}
}

func TestRunReturnsErrorForFailingWriter(t *testing.T) {
	if err := run(errWriter{}); err == nil {
		t.Fatal("expected run to return an error for failing writer")
	}
}

func TestMainUsesInjectedWritersAndExit(t *testing.T) {
	origStdout := stdout
	origStderr := stderr
	origExit := exitFunc
	t.Cleanup(func() {
		stdout = origStdout
		stderr = origStderr
		exitFunc = origExit
	})

	var out bytes.Buffer
	stdout = &out
	stderr = io.Discard

	var code int
	exitFunc = func(c int) {
		code = c
	}

	main()

	if code != 0 {
		t.Fatalf("expected exit code 0 from main, got %d", code)
	}

	if !strings.Contains(out.String(), "openapi: 3.1.0") {
		t.Fatalf("expected openapi document output, got: %s", out.String())
	}
}
