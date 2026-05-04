package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/client"
	"github.com/Bizarre-Industries/helling/apps/helling-cli/internal/config"
)

func TestClient_DoBearerAndRefreshCookieRotation(t *testing.T) {
	var sawAuthz, sawCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuthz = r.Header.Get("Authorization")
		sawCookie = r.Header.Get("Cookie")
		http.SetCookie(w, &http.Cookie{Name: "helling_session", Value: "rotated"})
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	prof := config.Profile{API: srv.URL, AccessToken: "jwt.x", RefreshCookie: "helling_session=old"}
	c, err := client.New(&prof, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.Do(context.Background(), "GET", "/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":true}` {
		t.Fatalf("body = %q", string(raw))
	}
	if sawAuthz != "Bearer jwt.x" {
		t.Errorf("authz = %q", sawAuthz)
	}
	if sawCookie != "helling_session=old" {
		t.Errorf("cookie = %q", sawCookie)
	}
	if c.RefreshCookie() != "helling_session=rotated" {
		t.Errorf("rotation not captured: %q", c.RefreshCookie())
	}
}

func TestClient_PostJSONBody(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	prof := config.Profile{API: srv.URL}
	c, _ := client.New(&prof, "")
	_, err := c.Do(context.Background(), "POST", "/submit", map[string]string{"a": "b"})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["a"] != "b" {
		t.Fatalf("body not sent: %+v", gotBody)
	}
}

func TestClient_ErrorEnvelopeIsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"title":"Unauthorized","status":401,"detail":"AUTH_BAD"}`))
	}))
	t.Cleanup(srv.Close)

	prof := config.Profile{API: srv.URL}
	c, _ := client.New(&prof, "")
	_, err := c.Do(context.Background(), "GET", "/x", nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.Status != 401 || apiErr.Detail != "AUTH_BAD" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestClient_HellingErrorEnvelopeIsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"already_setup","message":"admin user already exists"}`))
	}))
	t.Cleanup(srv.Close)

	prof := config.Profile{API: srv.URL}
	c, _ := client.New(&prof, "")
	_, err := c.Do(context.Background(), "POST", "/auth/setup", nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.Status != http.StatusConflict || apiErr.Detail != "already_setup: admin user already exists" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestClient_NewRequiresAPI(t *testing.T) {
	if _, err := client.New(&config.Profile{}, ""); err == nil {
		t.Fatal("expected error when API empty")
	}
}

func TestClient_ExplicitBaseURLOverridesProfile(t *testing.T) {
	c, err := client.New(&config.Profile{API: "http://should-be-ignored"}, "http://127.0.0.1:12345")
	if err != nil {
		t.Fatal(err)
	}
	if c.API() != "http://127.0.0.1:12345" {
		t.Fatalf("API() = %q", c.API())
	}
}

func TestClient_HTTPUnixEndpoint(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "api.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})}
	t.Cleanup(func() { _ = srv.Close() })
	go func() { _ = srv.Serve(ln) }()

	c, err := client.New(&config.Profile{}, "http+unix://"+socketPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.Do(context.Background(), "GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"ok":true}` {
		t.Fatalf("body = %q", raw)
	}
}
