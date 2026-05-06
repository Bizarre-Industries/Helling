package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsEndpointExposesPrometheusBaseline(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	client := ts.Client()
	resp := doGet(t, client, ts.URL+"/healthz")
	_ = resp.Body.Close()

	resp = doGet(t, client, ts.URL+"/metrics")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /metrics status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("Content-Type = %q", got)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	body := string(bodyBytes)
	for _, want := range []string{
		"# TYPE helling_api_requests_total counter",
		`helling_api_requests_total{method="GET",path="/healthz",status="200"}`,
		"# TYPE helling_api_request_duration_seconds histogram",
		"# TYPE helling_goroutines gauge",
		"# TYPE helling_open_connections gauge",
		"# TYPE helling_db_size_bytes gauge",
		"# TYPE helling_upstream_metrics_scrape_success gauge",
		`helling_upstream_metrics_scrape_success{upstream="incus"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
}

func TestMetricsIncludesProxiedIncusMetrics(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServerWithConfig(t, func(cfg *Config) {
		cfg.IncusMetrics = func(_ context.Context) (string, error) {
			return "# HELP incus_instances_total Instances known to Incus.\n# TYPE incus_instances_total gauge\nincus_instances_total 2", nil
		}
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := doGet(t, ts.Client(), ts.URL+"/metrics")
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	body := string(bodyBytes)
	for _, want := range []string{
		`helling_upstream_metrics_scrape_success{upstream="incus"} 1`,
		"# BEGIN proxied Incus /1.0/metrics",
		"incus_instances_total 2",
		"# END proxied Incus /1.0/metrics",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
}

func TestMetricsScrapeFailureKeepsHellingMetricsAvailable(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServerWithConfig(t, func(cfg *Config) {
		cfg.IncusMetrics = func(_ context.Context) (string, error) {
			return "", errors.New("incus unavailable")
		}
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := doGet(t, ts.Client(), ts.URL+"/metrics")
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	body := string(bodyBytes)
	for _, want := range []string{
		"# TYPE helling_api_requests_total counter",
		`helling_upstream_metrics_scrape_success{upstream="incus"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "# BEGIN proxied Incus /1.0/metrics") {
		t.Fatalf("failed Incus scrape unexpectedly appended upstream block:\n%s", body)
	}
}

func TestMetricsRecordsFiveHundredErrors(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	resp := doGet(t, ts.Client(), ts.URL+"/api/v1/system/diagnostics")
	_ = resp.Body.Close()

	// Unmatched paths are 404s and should not increment the 5xx error counter.
	resp = doGet(t, ts.Client(), ts.URL+"/missing")
	_ = resp.Body.Close()

	srv.metrics.record("GET", "/forced", http.StatusInternalServerError, 0)
	resp = doGet(t, ts.Client(), ts.URL+"/metrics")
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, `helling_api_errors_total{method="GET",path="/forced",status="500"} 1`) {
		t.Fatalf("metrics body missing forced 5xx error counter:\n%s", body)
	}
	if strings.Contains(body, `helling_api_errors_total{method="GET",path="unmatched",status="404"}`) {
		t.Fatalf("404 unexpectedly counted as 5xx error:\n%s", body)
	}
}

func doGet(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("new GET request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}
