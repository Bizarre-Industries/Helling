package store

// Kubernetes clusters CRUD. CAPN-provisioned K8s on Incus VMs.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// K8sCluster mirrors a row in the k8s_clusters table.
type K8sCluster struct {
	ID            string
	UserID        int64
	Name          string
	Version       string
	ControlPlanes int
	Workers       int
	Status        string
	Kubeconfig    *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateK8sCluster inserts a new cluster and returns it.
func (s *Store) CreateK8sCluster(ctx context.Context, userID int64, name, version string, controlPlanes, workers int) (K8sCluster, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return K8sCluster{}, fmt.Errorf("generating cluster id: %w", err)
	}
	now := time.Now().UTC()
	c := K8sCluster{
		ID:            id.String(),
		UserID:        userID,
		Name:          name,
		Version:       version,
		ControlPlanes: controlPlanes,
		Workers:       workers,
		Status:        "provisioning",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO k8s_clusters (id, user_id, name, version, control_planes, workers, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.Name, c.Version, c.ControlPlanes, c.Workers, c.Status, now.Unix(), now.Unix(),
	)
	if err != nil {
		return K8sCluster{}, fmt.Errorf("inserting k8s cluster: %w", err)
	}
	return c, nil
}

// GetK8sCluster returns a cluster by name, or ErrNotFound.
func (s *Store) GetK8sCluster(ctx context.Context, name string) (K8sCluster, error) {
	var c K8sCluster
	var createdAt, updatedAt int64
	var kubeconfig sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, version, control_planes, workers, status, kubeconfig, created_at, updated_at
		 FROM k8s_clusters WHERE name = ?`, name,
	).Scan(&c.ID, &c.UserID, &c.Name, &c.Version, &c.ControlPlanes, &c.Workers, &c.Status, &kubeconfig, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return K8sCluster{}, ErrNotFound
	}
	if err != nil {
		return K8sCluster{}, fmt.Errorf("loading k8s cluster: %w", err)
	}
	if kubeconfig.Valid {
		c.Kubeconfig = &kubeconfig.String
	}
	c.CreatedAt = time.Unix(createdAt, 0).UTC()
	c.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return c, nil
}

// ListK8sClusters returns all clusters.
func (s *Store) ListK8sClusters(ctx context.Context) ([]K8sCluster, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, version, control_planes, workers, status, kubeconfig, created_at, updated_at
		 FROM k8s_clusters ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing k8s clusters: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var clusters []K8sCluster
	for rows.Next() {
		var c K8sCluster
		var createdAt, updatedAt int64
		var kubeconfig sql.NullString
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Version, &c.ControlPlanes, &c.Workers, &c.Status, &kubeconfig, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning k8s cluster: %w", err)
		}
		if kubeconfig.Valid {
			c.Kubeconfig = &kubeconfig.String
		}
		c.CreatedAt = time.Unix(createdAt, 0).UTC()
		c.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

// UpdateK8sClusterStatus updates the status and optionally the kubeconfig.
func (s *Store) UpdateK8sClusterStatus(ctx context.Context, name, status string, kubeconfig *string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx,
		`UPDATE k8s_clusters SET status = ?, kubeconfig = ?, updated_at = ? WHERE name = ?`,
		status, kubeconfig, now, name,
	)
	if err != nil {
		return fmt.Errorf("updating k8s cluster status: %w", err)
	}
	return nil
}

// UpdateK8sClusterScale updates worker count.
func (s *Store) UpdateK8sClusterScale(ctx context.Context, name string, workers int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE k8s_clusters SET workers = ?, updated_at = ? WHERE name = ?`,
		workers, time.Now().Unix(), name,
	)
	if err != nil {
		return fmt.Errorf("scaling k8s cluster: %w", err)
	}
	return nil
}

// DeleteK8sCluster removes a cluster by name. Idempotent.
func (s *Store) DeleteK8sCluster(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM k8s_clusters WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("deleting k8s cluster: %w", err)
	}
	return nil
}
