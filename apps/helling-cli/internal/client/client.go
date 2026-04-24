// Package client is a thin HTTP client for the helling CLI. It speaks to
// hellingd over its listen address, injecting the Helling bearer token and
// (optionally) the helling_refresh cookie.
//
// A richer code-generated client (ADR-043 oapi-codegen) will eventually
// supersede this module; today's implementation stays small and explicit so
// the CLI can ship ahead of the full contract regeneration.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/helling-cli/internal/config"
)

// Client wraps http.Client with helling-specific headers + error handling.
type Client struct {
	api    string // base URL, e.g. http://127.0.0.1:8080
	http   *http.Client
	bearer string // access JWT or helling_* API token
	cookie string // helling_refresh=<value> or ""
}

// New builds a Client. baseURL overrides prof.API when non-empty.
func New(prof *config.Profile, baseURL string) (*Client, error) {
	api := baseURL
	if api == "" {
		api = prof.API
	}
	if api == "" {
		return nil, errors.New("client: no API endpoint configured (set -api or run 'helling auth login')")
	}
	if _, err := url.Parse(api); err != nil {
		return nil, fmt.Errorf("client: parse %q: %w", api, err)
	}
	return &Client{
		api:    strings.TrimRight(api, "/"),
		http:   &http.Client{Timeout: 30 * time.Second},
		bearer: prof.Bearer(),
		cookie: prof.RefreshCookie,
	}, nil
}

// APIError is the decoded Helling error envelope.
type APIError struct {
	Status int
	Detail string
	Body   string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("hellingd %d: %s", e.Status, e.Detail)
	}
	return fmt.Sprintf("hellingd %d: %s", e.Status, e.Body)
}

// Do performs a request, returning the response body on 2xx or an APIError
// otherwise. body may be nil; when non-nil it is JSON-marshaled.
func (c *Client) Do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("client: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.api+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		for _, sc := range resp.Cookies() {
			if sc.Name == "helling_refresh" {
				c.cookie = sc.Name + "=" + sc.Value
				break
			}
		}
		return raw, nil
	}

	var envelope struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
	}
	_ = json.Unmarshal(raw, &envelope)
	detail := envelope.Detail
	if detail == "" {
		detail = envelope.Title
	}
	return nil, &APIError{Status: resp.StatusCode, Detail: detail, Body: string(raw)}
}

// RefreshCookie returns the current refresh cookie after any rotation.
func (c *Client) RefreshCookie() string { return c.cookie }

// Bearer returns the current bearer token.
func (c *Client) Bearer() string { return c.bearer }

// API returns the configured endpoint.
func (c *Client) API() string { return c.api }
