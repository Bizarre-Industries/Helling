// Package poller mirrors Incus operation state into hellingd's operations
// table. It runs as a single background goroutine started from main.go and
// exits cleanly when its context is canceled.
package poller

import (
	"context"
	"log/slog"
	"time"

	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/incus"
	"github.com/Bizarre-Industries/helling/apps/hellingd/internal/store"
)

// Run blocks until ctx is done. Every interval it scans active operations
// (pending|running) and asks Incus for their current state, writing the
// result back to the store. A nil incus client is treated as "skip" — the
// poller stays alive but does no work, so a degraded daemon still ticks.
func Run(ctx context.Context, st *store.Store, c incus.Client, logger *slog.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c == nil {
				continue
			}
			tick(ctx, st, c, logger)
		}
	}
}

func tick(ctx context.Context, st *store.Store, c incus.Client, logger *slog.Logger) {
	ops, err := st.ListActiveOperations(ctx)
	if err != nil {
		logger.WarnContext(ctx, "poller: list active operations", slog.Any("err", err))
		return
	}
	for i := range ops {
		op := &ops[i]
		if op.IncusOpID == "" {
			// No upstream id to track — bookkeeping-only operation. Mark as success.
			if err := st.UpdateOperationStatus(ctx, op.ID, store.OpStatusSuccess, ""); err != nil {
				logger.WarnContext(ctx, "poller: marking bookkeeping op success", slog.String("op", op.ID), slog.Any("err", err))
			}
			continue
		}

		status, errMsg, err := c.GetOperation(ctx, op.IncusOpID)
		if err != nil {
			logger.WarnContext(ctx, "poller: get incus operation",
				slog.String("op", op.ID),
				slog.String("incus_op", op.IncusOpID),
				slog.Any("err", err),
			)
			continue
		}

		var next store.OperationStatus
		switch status {
		case incus.OpRunning:
			next = store.OpStatusRunning
		case incus.OpSuccess:
			next = store.OpStatusSuccess
		case incus.OpFailure:
			next = store.OpStatusFailure
		case incus.OpCancelled:
			next = store.OpStatusCancelled
		default:
			continue
		}

		if next == op.Status && errMsg == op.Error {
			continue
		}
		if err := st.UpdateOperationStatus(ctx, op.ID, next, errMsg); err != nil {
			logger.WarnContext(ctx, "poller: update operation",
				slog.String("op", op.ID),
				slog.String("status", string(next)),
				slog.Any("err", err),
			)
		}
	}
}
