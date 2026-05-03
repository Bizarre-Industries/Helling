// Package main is the entrypoint for the hellingd backend daemon.
//
// hellingd listens on a Unix socket only. It is never directly exposed
// to the network. All public traffic flows through helling-proxy, which
// terminates TLS and forwards to hellingd over the local socket.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/config"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/server"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// Build-time variables, populated via -ldflags by the Makefile.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "/etc/helling/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintf(os.Stdout, "hellingd %s (commit %s, built %s)\n", version, commit, buildTime)
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg.Log)
	slog.SetDefault(logger)

	logger.Info("starting hellingd",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("socket", cfg.Server.SocketPath),
		slog.String("state_dir", cfg.StateDir),
	)

	st, err := store.Open(cfg.StateDir)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			logger.Warn("closing store", slog.Any("err", cerr))
		}
	}()

	if err := st.Migrate(context.Background()); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	srv, err := server.New(server.Config{
		Store:   st,
		Logger:  logger,
		Version: server.VersionInfo{Version: version, Commit: commit, BuildTime: buildTime},
	})
	if err != nil {
		return fmt.Errorf("building server: %w", err)
	}

	listener, err := newSocketListener(cfg.Server.SocketPath, cfg.Server.SocketGroup, cfg.Server.SocketMode)
	if err != nil {
		return fmt.Errorf("creating socket listener: %w", err)
	}

	httpServer := &http.Server{
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "socket", cfg.Server.SocketPath)
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("shutdown complete")
	return nil
}

func newLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}

// newSocketListener creates a Unix socket listener with the requested
// permissions. It removes any stale socket file before binding.
func newSocketListener(path, group string, mode os.FileMode) (net.Listener, error) {
	// Remove stale socket from a previous run. systemd will pass an existing
	// listener via socket activation in the future; we ignore that here.
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("removing stale socket: %w", err)
		}
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(path, mode); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}

	// Group ownership is set via Chown in a real install; skipped here because
	// it requires privilege at startup. A systemd unit handles it via SocketUser=
	// and SocketGroup= directives. See deploy/systemd/helling.socket (TODO).
	_ = group

	return listener, nil
}
