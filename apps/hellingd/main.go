// Package main starts the hellingd daemon process.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
	httpserver "github.com/Bizarre-Industries/Helling/apps/hellingd/internal/http"
)

const (
	defaultAddr = ":8080"
	defaultDSN  = "file:/var/lib/helling/helling.db?cache=shared"
)

var defaultServe = func(server *http.Server) error {
	return server.ListenAndServe()
}

var (
	exitFunc           = os.Exit
	stderr   io.Writer = os.Stderr
	openDB             = db.Open
	osArgs             = os.Args
)

func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}

// runConfig holds parsed CLI flags, kept small so tests can synthesize one
// without invoking flag.Parse directly.
type runConfig struct {
	addr        string
	dsn         string
	migrateOnly bool
}

func parseFlags(args []string, errOut io.Writer) (runConfig, error) {
	fs := flag.NewFlagSet("hellingd", flag.ContinueOnError)
	fs.SetOutput(errOut)

	cfg := runConfig{}
	fs.StringVar(&cfg.addr, "addr", defaultAddr, "listen address")
	fs.StringVar(&cfg.dsn, "db", defaultDSN, "SQLite DSN (modernc.org/sqlite)")
	fs.BoolVar(&cfg.migrateOnly, "migrate-only", false, "apply migrations and exit")

	if err := fs.Parse(args); err != nil {
		return runConfig{}, err
	}
	return cfg, nil
}

func run(logger *slog.Logger, cfg runConfig, serve func(*http.Server) error) int {
	ctx := context.Background()

	pool, err := openDB(ctx, cfg.dsn)
	if err != nil {
		logger.Error("open db", slog.Any("err", err))
		return 1
	}
	defer func() { _ = pool.Close() }()

	logger.Info("db ready",
		slog.String("dsn", cfg.dsn),
		slog.Bool("migrate_only", cfg.migrateOnly),
	)

	if cfg.migrateOnly {
		return 0
	}

	mux := httpserver.NewMux()

	server := &http.Server{
		Addr:              cfg.addr,
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

// Keep database/sql imported for symbol stability across test stubs of openDB.
var _ = (*sql.DB)(nil)

func main() {
	logger := newLogger(stderr)

	cfg, err := parseFlags(osArgs[1:], stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			exitFunc(0)
			return
		}
		exitFunc(2)
		return
	}

	exitFunc(run(logger, cfg, defaultServe))
}
