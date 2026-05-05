package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FirewallRule mirrors one Helling-managed host firewall rule.
type FirewallRule struct {
	ID              string
	UserID          int64
	Direction       string
	Action          string
	Protocol        string
	SourceCIDR      *string
	DestinationCIDR *string
	DestinationPort *int
	Enabled         bool
	Comment         *string
	NFTComment      string
	NFTHandle       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// FirewallRuleInput contains validated fields for creating a host firewall rule.
type FirewallRuleInput struct {
	Direction       string
	Action          string
	Protocol        string
	SourceCIDR      *string
	DestinationCIDR *string
	DestinationPort *int
	Enabled         bool
	Comment         *string
}

// CreateFirewallRule stores a Helling-managed host firewall rule.
func (s *Store) CreateFirewallRule(ctx context.Context, userID int64, in *FirewallRuleInput) (FirewallRule, error) {
	if in == nil {
		return FirewallRule{}, errors.New("firewall rule input required")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return FirewallRule{}, fmt.Errorf("generating firewall rule id: %w", err)
	}
	now := time.Now().UTC()
	rule := FirewallRule{
		ID:              id.String(),
		UserID:          userID,
		Direction:       in.Direction,
		Action:          in.Action,
		Protocol:        in.Protocol,
		SourceCIDR:      in.SourceCIDR,
		DestinationCIDR: in.DestinationCIDR,
		DestinationPort: in.DestinationPort,
		Enabled:         in.Enabled,
		Comment:         in.Comment,
		NFTComment:      "helling:" + id.String(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	enabled := 0
	if rule.Enabled {
		enabled = 1
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO firewall_host_rules
		 (id, user_id, direction, action, protocol, source_cidr, destination_cidr, destination_port, enabled, comment, nft_comment, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.UserID, rule.Direction, rule.Action, rule.Protocol, nullableString(rule.SourceCIDR),
		nullableString(rule.DestinationCIDR), nullableInt(rule.DestinationPort), enabled, nullableString(rule.Comment),
		rule.NFTComment, now.Unix(), now.Unix(),
	)
	if err != nil {
		return FirewallRule{}, fmt.Errorf("inserting firewall rule: %w", err)
	}
	return rule, nil
}

// ListFirewallRules returns all Helling-managed host firewall rules.
func (s *Store) ListFirewallRules(ctx context.Context) ([]FirewallRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, direction, action, protocol, source_cidr, destination_cidr, destination_port, enabled, comment, nft_comment, nft_handle, created_at, updated_at
		 FROM firewall_host_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing firewall rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []FirewallRule
	for rows.Next() {
		rule, err := scanFirewallRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

// GetFirewallRule returns a Helling-managed host firewall rule by id.
func (s *Store) GetFirewallRule(ctx context.Context, id string) (FirewallRule, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, direction, action, protocol, source_cidr, destination_cidr, destination_port, enabled, comment, nft_comment, nft_handle, created_at, updated_at
		 FROM firewall_host_rules WHERE id = ?`,
		id,
	)
	rule, err := scanFirewallRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return FirewallRule{}, ErrNotFound
	}
	if err != nil {
		return FirewallRule{}, err
	}
	return rule, nil
}

// DeleteFirewallRule deletes a Helling-managed host firewall rule by id.
func (s *Store) DeleteFirewallRule(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM firewall_host_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting firewall rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking firewall delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateFirewallRuleHandle stores the nft handle observed after apply.
func (s *Store) UpdateFirewallRuleHandle(ctx context.Context, id string, handle int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE firewall_host_rules SET nft_handle = ?, updated_at = ? WHERE id = ?`,
		handle, time.Now().UTC().Unix(), id,
	)
	if err != nil {
		return fmt.Errorf("updating firewall rule handle: %w", err)
	}
	return nil
}

type firewallScanner interface {
	Scan(dest ...any) error
}

func scanFirewallRule(rows firewallScanner) (FirewallRule, error) {
	var r FirewallRule
	var source, dest, comment sql.NullString
	var port, handle sql.NullInt64
	var enabled int
	var createdAt, updatedAt int64
	if err := rows.Scan(&r.ID, &r.UserID, &r.Direction, &r.Action, &r.Protocol, &source, &dest, &port, &enabled, &comment, &r.NFTComment, &handle, &createdAt, &updatedAt); err != nil {
		return FirewallRule{}, fmt.Errorf("scanning firewall rule: %w", err)
	}
	r.SourceCIDR = ptrString(source)
	r.DestinationCIDR = ptrString(dest)
	if port.Valid {
		p := int(port.Int64)
		r.DestinationPort = &p
	}
	r.Enabled = enabled != 0
	r.Comment = ptrString(comment)
	if handle.Valid {
		h := handle.Int64
		r.NFTHandle = &h
	}
	r.CreatedAt = time.Unix(createdAt, 0).UTC()
	r.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return r, nil
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func ptrString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}
