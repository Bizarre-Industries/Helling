// Package client is a thin HTTP client for the helling CLI. It speaks to
// hellingd over its listen address, injecting the Helling bearer token and
// (optionally) the helling_session cookie.
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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

// Client wraps http.Client with helling-specific headers + error handling.
type Client struct {
	api    string // base URL, e.g. http://127.0.0.1:8080
	http   *http.Client
	bearer string // access JWT or helling_* API token
	cookie string // helling_session=<value> or ""
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
	httpClient := &http.Client{Timeout: 30 * time.Second}
	if socketPath, ok := strings.CutPrefix(api, "http+unix://"); ok {
		if socketPath == "" || !strings.HasPrefix(socketPath, "/") {
			return nil, fmt.Errorf("client: invalid http+unix endpoint %q", api)
		}
		httpClient.Transport = unixTransport(socketPath)
		api = "http://helling"
	} else if _, err := url.Parse(api); err != nil {
		return nil, fmt.Errorf("client: parse %q: %w", api, err)
	}
	return &Client{
		api:    strings.TrimRight(api, "/"),
		http:   httpClient,
		bearer: prof.Bearer(),
		cookie: prof.RefreshCookie,
	}, nil
}

func unixTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
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
	bodyReader, hasBody, err := encodeRequestBody(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.api+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if hasBody {
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
		c.captureSessionCookie(resp)
		return raw, nil
	}

	return nil, decodeAPIError(resp.StatusCode, raw)
}

func encodeRequestBody(body any) (io.Reader, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, false, fmt.Errorf("client: marshal body: %w", err)
	}
	return bytes.NewReader(buf), true, nil
}

func (c *Client) captureSessionCookie(resp *http.Response) {
	for _, sc := range resp.Cookies() {
		if sc.Name == "helling_session" {
			c.cookie = sc.Name + "=" + sc.Value
			return
		}
	}
}

func decodeAPIError(status int, raw []byte) error {
	var envelope struct {
		Detail  string `json:"detail"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	_ = json.Unmarshal(raw, &envelope)
	detail := envelope.Detail
	if detail == "" {
		detail = envelope.Title
	}
	if detail == "" {
		detail = envelope.Message
	}
	if envelope.Code != "" && detail != "" {
		detail = envelope.Code + ": " + detail
	} else if envelope.Code != "" {
		detail = envelope.Code
	}
	return &APIError{Status: status, Detail: detail, Body: string(raw)}
}

// RefreshCookie returns the current session cookie after any login.
func (c *Client) RefreshCookie() string { return c.cookie }

// Bearer returns the current bearer token.
func (c *Client) Bearer() string { return c.bearer }

// API returns the configured endpoint.
func (c *Client) API() string { return c.api }
