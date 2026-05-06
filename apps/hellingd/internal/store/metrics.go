package store

import (
	"context"
	"fmt"
)

// DBSizeBytes returns SQLite's current page_count * page_size estimate.
func (s *Store) DBSizeBytes(ctx context.Context) (int64, error) {
	var pageCount, pageSize int64
	if err := s.db.QueryRowContext(ctx, `PRAGMA page_count`).Scan(&pageCount); err != nil {
		return 0, fmt.Errorf("querying sqlite page_count: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `PRAGMA page_size`).Scan(&pageSize); err != nil {
		return 0, fmt.Errorf("querying sqlite page_size: %w", err)
	}
	return pageCount * pageSize, nil
}
