package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRealWebhooks_Lifecycle(t *testing.T) {
	srv := spinUp(t)
	access := setupAdmin(t, srv, "admin", "whpw1234567890")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	createResp := bearerPost(t, srv, "/api/v1/webhooks", access, map[string]any{
		"name":   "ci-webhook",
		"url":    upstream.URL,
		"secret": "abcdefghijklmnop",
		"events": []string{"instance.created"},
	})
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d body=%s", createResp.StatusCode, mustReadBody(createResp))
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	readJSON(t, createResp, &created)
	if created.Data.ID == "" {
		t.Fatal("missing id")
	}

	listResp := bearerGet(t, srv, "/api/v1/webhooks", access)
	defer func() { _ = listResp.Body.Close() }()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", listResp.StatusCode)
	}
	if !strings.Contains(mustReadBody(listResp), "ci-webhook") {
		t.Fatal("list missing ci-webhook")
	}

	body, _ := json.Marshal(map[string]any{"name": "ci-renamed"})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPatch,
		srv.URL+"/api/v1/webhooks/"+created.Data.ID, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	patch, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = patch.Body.Close() }()
	if patch.StatusCode != http.StatusOK {
		t.Fatalf("update: %d body=%s", patch.StatusCode, mustReadBody(patch))
	}

	testResp := bearerPost(t, srv, "/api/v1/webhooks/"+created.Data.ID+"/test", access, struct{}{})
	defer func() { _ = testResp.Body.Close() }()
	if testResp.StatusCode != http.StatusOK {
		t.Fatalf("test: %d body=%s", testResp.StatusCode, mustReadBody(testResp))
	}
	if !strings.Contains(mustReadBody(testResp), `"delivered"`) {
		t.Fatal("delivery status missing")
	}

	delResp := bearerDelete(t, srv, "/api/v1/webhooks/"+created.Data.ID, access)
	defer func() { _ = delResp.Body.Close() }()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d", delResp.StatusCode)
	}
	missing := bearerGet(t, srv, "/api/v1/webhooks/"+created.Data.ID, access)
	defer func() { _ = missing.Body.Close() }()
	if missing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", missing.StatusCode)
	}
}

func TestRealWebhooks_RequiresAuth(t *testing.T) {
	srv := spinUp(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/v1/webhooks", http.NoBody)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
