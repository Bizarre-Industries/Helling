// Package main starts the hellingd daemon process.
package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	httpserver "github.com/Bizarre-Industries/Helling/apps/hellingd/internal/http"
)

const defaultAddr = ":8080"

var defaultServe = func(server *http.Server) error {
	return server.ListenAndServe()
}

var exitFunc = os.Exit

var stderr io.Writer = os.Stderr

func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}

func run(logger *slog.Logger, serve func(*http.Server) error) int {
	mux := httpserver.NewMux()

	server := &http.Server{
		Addr:              defaultAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("hellingd listening", slog.String("addr", server.Addr))
	if err := serve(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("hellingd stopped", slog.Any("err", err))
		return 1
	}

	return 0
}

func main() {
	logger := newLogger(stderr)
	exitFunc(run(logger, defaultServe))
}
