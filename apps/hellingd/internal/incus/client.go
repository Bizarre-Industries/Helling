// Package incus wraps the Incus HTTP API. All Incus interaction goes through
// this package; handlers and services depend on the Client interface so tests
// can swap in fakes without dialing a real socket.
package incus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Client is the narrow surface hellingd uses against Incus. Real implementation
// is realClient; tests use a fake.
type Client interface {
	ListInstances(ctx context.Context) ([]Instance, error)
	GetInstance(ctx context.Context, name string) (*Instance, error)
	CreateInstance(ctx context.Context, req InstanceCreate) (OperationHandle, error)
	UpdateInstanceState(ctx context.Context, name, action string, force bool, timeoutSec int) (OperationHandle, error)
	DeleteInstance(ctx context.Context, name string) (OperationHandle, error)
	GetOperation(ctx context.Context, id string) (OperationStatus, string, error)
}

// InstanceCreate is the narrow create payload used by Helling's v0.1 instance
// endpoint.
type InstanceCreate struct {
	Name  string
	Type  string
	Image string
	Start bool
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
// and a Wait method for synchronous flows.
type OperationHandle interface {
	ID() string
	Wait() error
}

// Connect dials the Incus Unix socket and returns a Client.
func Connect(socketPath string) (Client, error) {
	if socketPath == "" {
		socketPath = "/var/lib/incus/unix.socket.user"
	}
	return &realClient{
		http: &http.Client{Transport: UnixTransport(socketPath)},
	}, nil
}

type realClient struct {
	http *http.Client
}

func (c *realClient) ListInstances(ctx context.Context) ([]Instance, error) {
	var out []Instance
	if err := c.do(ctx, http.MethodGet, "/1.0/instances?recursion=1", nil, &out); err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	return out, nil
}

func (c *realClient) GetInstance(ctx context.Context, name string) (*Instance, error) {
	var out Instance
	if err := c.do(ctx, http.MethodGet, "/1.0/instances/"+pathEscape(name), nil, &out); err != nil {
		return nil, fmt.Errorf("getting instance %q: %w", name, err)
	}
	return &out, nil
}

func (c *realClient) CreateInstance(ctx context.Context, req InstanceCreate) (OperationHandle, error) {
	body := map[string]any{
		"name": req.Name,
		"type": req.Type,
		"source": map[string]any{
			"type":  "image",
			"alias": req.Image,
		},
		"start": req.Start,
	}
	op, err := c.doOperation(ctx, http.MethodPost, "/1.0/instances", body)
	if err != nil {
		return nil, fmt.Errorf("creating instance %q: %w", req.Name, err)
	}
	return op, nil
}

func (c *realClient) UpdateInstanceState(ctx context.Context, name, action string, force bool, timeoutSec int) (OperationHandle, error) {
	body := map[string]any{
		"action":   action,
		"force":    force,
		"timeout":  timeoutSec,
		"stateful": false,
	}
	op, err := c.doOperation(ctx, http.MethodPut, "/1.0/instances/"+pathEscape(name)+"/state", body)
	if err != nil {
		return nil, fmt.Errorf("updating instance %q state to %q: %w", name, action, err)
	}
	return op, nil
}

func (c *realClient) DeleteInstance(ctx context.Context, name string) (OperationHandle, error) {
	op, err := c.doOperation(ctx, http.MethodDelete, "/1.0/instances/"+pathEscape(name), nil)
	if err != nil {
		return nil, fmt.Errorf("deleting instance %q: %w", name, err)
	}
	return op, nil
}

// GetOperation queries the current state of a known Incus operation by id.
// Returns OpUnknown when the id is unknown to Incus.
func (c *realClient) GetOperation(ctx context.Context, id string) (OperationStatus, string, error) {
	if id == "" {
		return OpUnknown, "", nil
	}
	var op incusOperation
	if err := c.do(ctx, http.MethodGet, "/1.0/operations/"+pathEscape(id), nil, &op); err != nil {
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

func (c *realClient) doOperation(ctx context.Context, method, requestPath string, body any) (OperationHandle, error) {
	var op incusOperation
	location, err := c.doWithLocation(ctx, method, requestPath, body, &op)
	if err != nil {
		return nil, err
	}
	id := op.ID
	if id == "" {
		id = operationIDFromLocation(location)
	}
	return operationHandle{id: id}, nil
}

func (c *realClient) do(ctx context.Context, method, requestPath string, body any, out any) error {
	_, err := c.doWithLocation(ctx, method, requestPath, body, out)
	return err
}

func (c *realClient) doWithLocation(ctx context.Context, method, requestPath string, body any, out any) (string, error) {
	req, err := newRequest(ctx, method, requestPath, body)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	return decodeResponse(resp, out)
}

func newRequest(ctx context.Context, method, requestPath string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://incus"+requestPath, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func decodeResponse(resp *http.Response, out any) (string, error) {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", incusHTTPError{status: resp.StatusCode, body: raw}
	}

	if out == nil || len(raw) == 0 {
		return resp.Header.Get("Location"), nil
	}
	var envelope incusResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("decode response envelope: %w", err)
	}
	if len(envelope.Metadata) == 0 || string(envelope.Metadata) == "null" {
		return resp.Header.Get("Location"), nil
	}
	if err := json.Unmarshal(envelope.Metadata, out); err != nil {
		var operationURL string
		if strErr := json.Unmarshal(envelope.Metadata, &operationURL); strErr == nil {
			if op, ok := out.(*incusOperation); ok {
				op.ID = operationIDFromLocation(operationURL)
				return resp.Header.Get("Location"), nil
			}
		}
		return "", fmt.Errorf("decode response metadata: %w", err)
	}
	return resp.Header.Get("Location"), nil
}

type incusResponse struct {
	Type       string          `json:"type"`
	Status     string          `json:"status"`
	StatusCode int             `json:"status_code"`
	Metadata   json.RawMessage `json:"metadata"`
	Error      string          `json:"error"`
}

type incusOperation struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Err    string `json:"err"`
}

type operationHandle struct {
	id string
}

func (h operationHandle) ID() string  { return h.id }
func (h operationHandle) Wait() error { return nil }

type incusHTTPError struct {
	status int
	body   []byte
}

func (e incusHTTPError) Error() string {
	var envelope incusResponse
	if err := json.Unmarshal(e.body, &envelope); err == nil && envelope.Error != "" {
		return fmt.Sprintf("incus http %d: %s", e.status, envelope.Error)
	}
	return fmt.Sprintf("incus http %d: %s", e.status, strings.TrimSpace(string(e.body)))
}

// UnixTransport returns an http.RoundTripper that dials a Unix socket.
func UnixTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			d.Timeout = 5 * time.Second
			return d.DialContext(ctx, "unix", socketPath)
		},
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

func pathEscape(s string) string {
	return url.PathEscape(s)
}

func operationIDFromLocation(location string) string {
	if location == "" {
		return ""
	}
	return path.Base(strings.TrimRight(location, "/"))
}
