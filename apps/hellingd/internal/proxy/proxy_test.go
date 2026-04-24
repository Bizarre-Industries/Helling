package proxy_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/auth"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/db"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/proxy"
	"github.com/Bizarre-Industries/Helling/apps/hellingd/internal/repo/authrepo"
)

// wsKeyFixture is an inert 16-byte base64 nonce used by the WebSocket
// handshake test. Real Sec-WebSocket-Key values are random and not secrets;
// gitleaks pattern-matches generic base64 strings, so we keep this in a
// named const with an allow-comment to make the intent explicit.
const wsKeyFixture = "AQIDBAUGBwgJCgsMDQ4PEA==" // gitleaks:allow

// newSvcWithAdmin boots an auth.Service with a local admin and returns the
// service + an access token for the admin.
func newSvcWithAdmin(t *testing.T) (svc *auth.Service, adminAccessToken string) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "proxy.db") + "?cache=shared"
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	_, priv, _ := auth.GenerateKey()
	signer := auth.NewSigner(priv, "hellingd-proxy-test", 15*time.Minute, 7*24*time.Hour, 30*time.Minute)
	svc = auth.NewService(authrepo.New(pool), signer, auth.Argon2idParams{})
	ident, err := svc.Setup(context.Background(), "admin", "proxypassword12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	return svc, ident.AccessToken
}

// fakeUpstream serves whatever JSON body the test prepared and echoes the
// user forwarded by the proxy. Always responds 200 OK.
func fakeUpstream(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if user := r.Header.Get("X-Forwarded-User"); user != "" {
			w.Header().Set("X-Upstream-Saw-User", user)
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestProxy_IncusForwardsWithBearer(t *testing.T) {
	svc, token := newSvcWithAdmin(t)
	upstream := fakeUpstream(t, `{"metadata":["/1.0/instances/foo"]}`)

	p, err := proxy.New(&proxy.Config{IncusURL: upstream.URL}, svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"metadata"`) {
		t.Fatalf("body = %q", string(body))
	}
	if resp.Header.Get("X-Proxy-Source") != "incus" {
		t.Errorf("missing X-Proxy-Source header")
	}
	if user := resp.Header.Get("X-Upstream-Saw-User"); user == "" {
		t.Errorf("upstream did not receive X-Forwarded-User")
	}
}

func TestProxy_Unauthenticated_Returns401(t *testing.T) {
	svc, _ := newSvcWithAdmin(t)
	upstream := fakeUpstream(t, "{}")
	p, _ := proxy.New(&proxy.Config{IncusURL: upstream.URL}, svc, nil)
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestProxy_WebSocketUpgradePassesThrough(t *testing.T) {
	svc, token := newSvcWithAdmin(t)
	// Raw TCP upstream so we can craft a precise 101 response that
	// httputil.ReverseProxy can forward via its Upgrade hijack path.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 4096)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	}()

	p, _ := proxy.New(&proxy.Config{IncusURL: "http://" + ln.Addr().String()}, svc, nil)
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances/x/console?type=vga", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", wsKeyFixture) // gitleaks:allow inert ws nonce
	req.Header.Set("Sec-WebSocket-Version", "13")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}
}

func TestProxy_NoUpstreamConfiguredReturns503(t *testing.T) {
	svc, token := newSvcWithAdmin(t)
	p, _ := proxy.New(&proxy.Config{}, svc, nil)
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	mux.Handle("/api/podman/", p.PodmanHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	for _, path := range []string{"/api/incus/1.0/instances", "/api/podman/libpod/containers/json"} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, edge.URL+path, http.NoBody)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want 503", path, resp.StatusCode)
		}
	}
}

func TestProxy_APITokenBearerAlsoWorks(t *testing.T) {
	svc, adminAccess := newSvcWithAdmin(t)
	claims, err := svc.Signer().Verify(adminAccess)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := svc.CreateAPIToken(context.Background(), claims.Subject, "bot", "read", 0)
	if err != nil {
		t.Fatal(err)
	}

	upstream := fakeUpstream(t, `{"ok":true}`)
	p, _ := proxy.New(&proxy.Config{IncusURL: upstream.URL}, svc, nil)
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+issued.Plaintext)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestConfigFromEnv_EmptyWhenUnset(t *testing.T) {
	t.Setenv("HELLING_INCUS_URL", "")
	t.Setenv("HELLING_PODMAN_SOCKET", "")
	cfg := proxy.ConfigFromEnv()
	if cfg.IncusURL != "" || cfg.PodmanSocket != "" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestConfigFromEnv_Populated(t *testing.T) {
	t.Setenv("HELLING_INCUS_URL", "https://127.0.0.1:8443")
	t.Setenv("HELLING_INCUS_CLIENT_CERT", "/etc/helling/incus/client.crt")
	t.Setenv("HELLING_INCUS_CLIENT_KEY", "/etc/helling/incus/client.key")
	t.Setenv("HELLING_PODMAN_SOCKET", "/run/podman/podman.sock")
	t.Setenv("HELLING_PROXY_INSECURE_SKIP_VERIFY", "1")
	cfg := proxy.ConfigFromEnv()
	if cfg.IncusURL == "" || cfg.PodmanSocket == "" || !cfg.InsecureSkipVerify {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestURLParseSanity(t *testing.T) {
	u, err := url.Parse("https://127.0.0.1:8443")
	if err != nil || u.Scheme != "https" {
		t.Fatal(err)
	}
}

func TestProxy_PodmanForwardsOverUnixSocket(t *testing.T) {
	// Unix socket paths have a 104-byte limit on macOS (108 on Linux). Use a
	// short relative path to stay under the cap, then chdir into the temp
	// dir so the relative path resolves.
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	sock := "pod.sock"
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"Names":["ct1"]}]`))
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	svc, token := newSvcWithAdmin(t)
	p, err := proxy.New(&proxy.Config{PodmanSocket: sock}, svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/podman/", p.PodmanHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/podman/libpod/containers/json", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"Names"`) {
		t.Fatalf("body = %q", string(body))
	}
	if resp.Header.Get("X-Proxy-Source") != "podman" {
		t.Errorf("missing X-Proxy-Source=podman")
	}
}

func TestProxy_IncusUpstreamDownYields502(t *testing.T) {
	svc, token := newSvcWithAdmin(t)
	p, err := proxy.New(&proxy.Config{IncusURL: "http://127.0.0.1:1"}, svc, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
}

func TestProxy_BadBearerToken_Returns401(t *testing.T) {
	svc, _ := newSvcWithAdmin(t)
	upstream := fakeUpstream(t, "{}")
	p, _ := proxy.New(&proxy.Config{IncusURL: upstream.URL}, svc, nil)
	mux := http.NewServeMux()
	mux.Handle("/api/incus/", p.IncusHandler())
	edge := httptest.NewServer(mux)
	t.Cleanup(edge.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		edge.URL+"/api/incus/1.0/instances", http.NoBody)
	req.Header.Set("Authorization", "Bearer garbage.jwt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestProxy_NewInvalidIncusURLErrors(t *testing.T) {
	if _, err := proxy.New(&proxy.Config{IncusURL: "://bad"}, nil, nil); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
