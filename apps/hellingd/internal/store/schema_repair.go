package store

import (
	"context"
	"fmt"
)

func (s *Store) repairV02Schema(ctx context.Context) error {
	if has, err := s.tableHasColumn(ctx, "event_outbox", "webhook_id"); err != nil {
		return err
	} else if !has {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE event_outbox ADD COLUMN webhook_id TEXT REFERENCES webhooks(id) ON DELETE CASCADE`); err != nil {
			return fmt.Errorf("repairing event_outbox.webhook_id: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE event_outbox SET webhook_id = substr(id, instr(id, ':') + 1) WHERE webhook_id IS NULL AND instr(id, ':') > 0`); err != nil {
			return fmt.Errorf("backfilling event_outbox.webhook_id: %w", err)
		}
	}
	if has, err := s.tableHasColumn(ctx, "incus_user_certs", "encrypted_cert_pem"); err != nil {
		return err
	} else if !has {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE incus_user_certs ADD COLUMN encrypted_cert_pem TEXT`); err != nil {
			return fmt.Errorf("repairing incus_user_certs.encrypted_cert_pem: %w", err)
		}
		if hasCertPEM, err := s.tableHasColumn(ctx, "incus_user_certs", "cert_pem"); err != nil {
			return err
		} else if hasCertPEM {
			if _, err := s.db.ExecContext(ctx, `UPDATE incus_user_certs SET encrypted_cert_pem = cert_pem WHERE encrypted_cert_pem IS NULL`); err != nil {
				return fmt.Errorf("backfilling incus_user_certs.encrypted_cert_pem: %w", err)
			}
		}
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_event_outbox_claim ON event_outbox(status, next_run_at, created_at, id)`); err != nil {
		return fmt.Errorf("repairing event_outbox claim index: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_event_outbox_stale_processing ON event_outbox(status, updated_at, id)`); err != nil {
		return fmt.Errorf("repairing event_outbox stale index: %w", err)
	}
	return nil
}

func (s *Store) tableHasColumn(ctx context.Context, table, column string) (bool, error) {
	var query string
	switch table {
	case "event_outbox":
		query = `PRAGMA table_info(event_outbox)`
	case "incus_user_certs":
		query = `PRAGMA table_info(incus_user_certs)`
	default:
		return false, fmt.Errorf("unsupported schema table %q", table)
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("reading %s schema: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("scanning %s schema: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("reading %s schema rows: %w", table, err)
	}
	return false, nil
}
