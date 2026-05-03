// Package incus wraps the Incus Go client. All Incus interaction goes through
// this package; handlers and services depend on the Client interface so tests
// can swap in fakes without dialing a real socket.
package incus

import (
	"context"
	"fmt"
	"strings"

	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

// Client is the narrow surface hellingd uses against Incus. Real implementation
// is realClient; tests use a fake.
type Client interface {
	ListInstances(ctx context.Context) ([]api.Instance, error)
	GetInstance(ctx context.Context, name string) (*api.Instance, error)
	CreateInstance(ctx context.Context, req api.InstancesPost) (OperationHandle, error)
	UpdateInstanceState(ctx context.Context, name, action string, force bool, timeoutSec int) (OperationHandle, error)
	DeleteInstance(ctx context.Context, name string) (OperationHandle, error)
	GetOperation(ctx context.Context, id string) (OperationStatus, string, error)
}

// OperationStatus is the lifecycle state of an Incus operation, mapped to our
// store.OperationStatus values upstream.
type OperationStatus string

// Mapped Incus operation lifecycle states.
const (
	OpUnknown   OperationStatus = "unknown"
	OpRunning   OperationStatus = "running"
	OpSuccess   OperationStatus = "success"
	OpFailure   OperationStatus = "failure"
	OpCancelled OperationStatus = "cancelled" //nolint:misspell // Incus uses British spelling
)

// OperationHandle is the minimal operation surface we need: an ID for tracking
// and a Wait method for synchronous flows. Mirrors incusclient.Operation but
// keeps our packages decoupled from the upstream type.
type OperationHandle interface {
	ID() string
	Wait() error
}

// Connect dials the Incus Unix socket and returns a Client.
func Connect(socketPath string) (Client, error) {
	if socketPath == "" {
		socketPath = "/var/lib/incus/unix.socket"
	}
	srv, err := incusclient.ConnectIncusUnix(socketPath, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to Incus at %s: %w", socketPath, err)
	}
	return &realClient{srv: srv}, nil
}

type realClient struct {
	srv incusclient.InstanceServer
}

func (c *realClient) ListInstances(_ context.Context) ([]api.Instance, error) {
	insts, err := c.srv.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	return insts, nil
}

func (c *realClient) GetInstance(_ context.Context, name string) (*api.Instance, error) {
	inst, _, err := c.srv.GetInstance(name)
	if err != nil {
		return nil, fmt.Errorf("getting instance %q: %w", name, err)
	}
	return inst, nil
}

//nolint:gocritic // upstream Incus client signature requires value receiver for InstancesPost
func (c *realClient) CreateInstance(_ context.Context, req api.InstancesPost) (OperationHandle, error) {
	op, err := c.srv.CreateInstance(req)
	if err != nil {
		return nil, fmt.Errorf("creating instance %q: %w", req.Name, err)
	}
	return &realOp{op: op}, nil
}

func (c *realClient) UpdateInstanceState(_ context.Context, name, action string, force bool, timeoutSec int) (OperationHandle, error) {
	op, err := c.srv.UpdateInstanceState(name, api.InstanceStatePut{
		Action:   action,
		Force:    force,
		Timeout:  timeoutSec,
		Stateful: false,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("updating instance %q state to %q: %w", name, action, err)
	}
	return &realOp{op: op}, nil
}

func (c *realClient) DeleteInstance(_ context.Context, name string) (OperationHandle, error) {
	op, err := c.srv.DeleteInstance(name)
	if err != nil {
		return nil, fmt.Errorf("deleting instance %q: %w", name, err)
	}
	return &realOp{op: op}, nil
}

// GetOperation queries the current state of a known Incus operation by id.
// Returns OpUnknown when the id is unknown to Incus (e.g. operation expired
// from Incus's in-memory ledger).
func (c *realClient) GetOperation(_ context.Context, id string) (OperationStatus, string, error) {
	if id == "" {
		return OpUnknown, "", nil
	}
	op, _, err := c.srv.GetOperation(id)
	if err != nil {
		return OpUnknown, "", fmt.Errorf("get incus operation %s: %w", id, err)
	}
	switch strings.ToLower(op.Status) {
	case "success":
		return OpSuccess, op.Err, nil
	case "failure":
		return OpFailure, op.Err, nil
	case "cancelled": //nolint:misspell // Incus reports British spelling
		return OpCancelled, op.Err, nil
	case "pending", "running":
		return OpRunning, op.Err, nil
	default:
		return OpUnknown, op.Err, nil
	}
}

type realOp struct {
	op incusclient.Operation
}

func (r *realOp) ID() string {
	if r.op == nil {
		return ""
	}
	return r.op.Get().ID
}

func (r *realOp) Wait() error {
	if r.op == nil {
		return nil
	}
	return r.op.Wait()
}
