package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestRealSystem_InfoAndHardware(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "sysrealpw12345")

	info := bearerGet(t, srv, "/api/v1/system/info", access)
	defer func() { _ = info.Body.Close() }()
	if info.StatusCode != http.StatusOK {
		t.Fatalf("info: %d body=%s", info.StatusCode, mustReadBody(info))
	}
	if !strings.Contains(mustReadBody(info), `"version"`) {
		t.Fatal("info body missing version")
	}

	hw := bearerGet(t, srv, "/api/v1/system/hardware", access)
	defer func() { _ = hw.Body.Close() }()
	if hw.StatusCode != http.StatusOK {
		t.Fatalf("hw: %d", hw.StatusCode)
	}
}

func TestRealSystem_ConfigPutGet(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "sysrealpw12345")

	miss := bearerGet(t, srv, "/api/v1/system/config/auth.session_inactivity_timeout", access)
	defer func() { _ = miss.Body.Close() }()
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", miss.StatusCode)
	}

	put := bearerPut(t, srv, "/api/v1/system/config/auth.session_inactivity_timeout", access, map[string]string{"value": "1800"})
	defer func() { _ = put.Body.Close() }()
	if put.StatusCode != http.StatusOK {
		t.Fatalf("put: %d body=%s", put.StatusCode, mustReadBody(put))
	}

	got := bearerGet(t, srv, "/api/v1/system/config/auth.session_inactivity_timeout", access)
	defer func() { _ = got.Body.Close() }()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("get: %d", got.StatusCode)
	}
	if !strings.Contains(mustReadBody(got), `"1800"`) {
		t.Fatal("get body missing value")
	}
}

func TestRealSystem_Diagnostics(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "sysrealpw12345")

	resp := bearerGet(t, srv, "/api/v1/system/diagnostics", access)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("diagnostics: %d", resp.StatusCode)
	}
	body := mustReadBody(resp)
	if !strings.Contains(body, `"db.reachable"`) || !strings.Contains(body, `"auth.signer"`) {
		t.Fatalf("diagnostics missing checks: %s", body)
	}
}

func TestRealSystem_UpgradeNoChange(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "sysrealpw12345")

	resp := bearerPost(t, srv, "/api/v1/system/upgrade", access, map[string]bool{"rollback": false})
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upgrade: %d", resp.StatusCode)
	}
	if !strings.Contains(mustReadBody(resp), `"no_change"`) {
		t.Fatal("expected no_change status")
	}
}

func TestRealSystem_Unauthenticated(t *testing.T) {
	srv := spinUp(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/v1/system/info", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
