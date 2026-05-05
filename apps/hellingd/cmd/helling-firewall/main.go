// Package main provides the narrow host firewall mutation helper.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/firewall"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := (firewall.Helper{Stderr: os.Stderr}).Handle(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "helling-firewall: %v\n", err)
		return 2
	}
	return 0
}
