package store

// BMC endpoints CRUD. IPMI/Redfish BMC management per docs/spec/architecture.md.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BMCEndpoint mirrors a row in the bmc_endpoints table.
type BMCEndpoint struct {
	ID        string
	UserID    int64
	Name      string
	Address   string
	Port      int
	Username  string
	Password  string
	Protocol  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateBMCEndpoint inserts a new BMC endpoint and returns it.
func (s *Store) CreateBMCEndpoint(ctx context.Context, userID int64, name, address string, port int, username, password, protocol string) (BMCEndpoint, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return BMCEndpoint{}, fmt.Errorf("generating bmc id: %w", err)
	}
	now := time.Now().UTC()
	b := BMCEndpoint{
		ID:        id.String(),
		UserID:    userID,
		Name:      name,
		Address:   address,
		Port:      port,
		Username:  username,
		Password:  password,
		Protocol:  protocol,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO bmc_endpoints (id, user_id, name, address, port, username, password, protocol, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.UserID, b.Name, b.Address, b.Port, b.Username, b.Password, b.Protocol, now.Unix(), now.Unix(),
	)
	if err != nil {
		return BMCEndpoint{}, fmt.Errorf("inserting bmc endpoint: %w", err)
	}
	return b, nil
}

// GetBMCEndpoint returns a BMC endpoint by id, or ErrNotFound.
func (s *Store) GetBMCEndpoint(ctx context.Context, id string) (BMCEndpoint, error) {
	var b BMCEndpoint
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, name, address, port, username, password, protocol, created_at, updated_at
		 FROM bmc_endpoints WHERE id = ?`, id,
	).Scan(&b.ID, &b.UserID, &b.Name, &b.Address, &b.Port, &b.Username, &b.Password, &b.Protocol, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return BMCEndpoint{}, ErrNotFound
	}
	if err != nil {
		return BMCEndpoint{}, fmt.Errorf("loading bmc endpoint: %w", err)
	}
	b.CreatedAt = time.Unix(createdAt, 0).UTC()
	b.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return b, nil
}

// ListBMCEndpoints returns all BMC endpoints.
func (s *Store) ListBMCEndpoints(ctx context.Context) ([]BMCEndpoint, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, address, port, username, password, protocol, created_at, updated_at
		 FROM bmc_endpoints ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing bmc endpoints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var endpoints []BMCEndpoint
	for rows.Next() {
		var b BMCEndpoint
		var createdAt, updatedAt int64
		if err := rows.Scan(&b.ID, &b.UserID, &b.Name, &b.Address, &b.Port, &b.Username, &b.Password, &b.Protocol, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning bmc endpoint: %w", err)
		}
		b.CreatedAt = time.Unix(createdAt, 0).UTC()
		b.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		endpoints = append(endpoints, b)
	}
	return endpoints, rows.Err()
}

// DeleteBMCEndpoint removes a BMC endpoint by id. Idempotent.
func (s *Store) DeleteBMCEndpoint(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM bmc_endpoints WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting bmc endpoint: %w", err)
	}
	return nil
}
