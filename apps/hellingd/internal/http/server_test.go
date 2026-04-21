package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAPIReturnsNonNil(t *testing.T) {
	mux := http.NewServeMux()
	if api := NewAPI(mux); api == nil {
		t.Fatal("expected NewAPI to return a non-nil huma.API")
	}
}

func TestNewMuxRegistersHealthRoute(t *testing.T) {
	mux := NewMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
