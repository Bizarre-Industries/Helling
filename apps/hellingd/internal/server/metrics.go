package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var apiDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type apiMetricKey struct {
	Method string
	Path   string
	Status string
}

type apiMetricSample struct {
	Count   uint64
	Errors  uint64
	Sum     float64
	Buckets []uint64
}

type metricsRegistry struct {
	mu  sync.Mutex
	api map[apiMetricKey]*apiMetricSample
}

func newMetricsRegistry() *metricsRegistry {
	return &metricsRegistry{api: make(map[apiMetricKey]*apiMetricSample)}
}

func (m *metricsRegistry) record(method, path string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	if path == "" {
		path = "unknown"
	}
	key := apiMetricKey{Method: method, Path: path, Status: strconv.Itoa(status)}
	seconds := duration.Seconds()

	m.mu.Lock()
	defer m.mu.Unlock()
	sample := m.api[key]
	if sample == nil {
		sample = &apiMetricSample{Buckets: make([]uint64, len(apiDurationBuckets))}
		m.api[key] = sample
	}
	sample.Count++
	if status >= 500 {
		sample.Errors++
	}
	sample.Sum += seconds
	for i, upper := range apiDurationBuckets {
		if seconds <= upper {
			sample.Buckets[i]++
		}
	}
}

func (m *metricsRegistry) snapshot() []struct {
	Key    apiMetricKey
	Sample apiMetricSample
} {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]struct {
		Key    apiMetricKey
		Sample apiMetricSample
	}, 0, len(m.api))
	for key, sample := range m.api {
		copySample := *sample
		copySample.Buckets = append([]uint64(nil), sample.Buckets...)
		out = append(out, struct {
			Key    apiMetricKey
			Sample apiMetricSample
		}{Key: key, Sample: copySample})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key.Path != out[j].Key.Path {
			return out[i].Key.Path < out[j].Key.Path
		}
		if out[i].Key.Method != out[j].Key.Method {
			return out[i].Key.Method < out[j].Key.Method
		}
		return out[i].Key.Status < out[j].Key.Status
	})
	return out
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		path := "unmatched"
		if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
			if pattern := routeCtx.RoutePattern(); pattern != "" {
				path = pattern
			}
		}
		s.metrics.record(r.Method, path, status, time.Since(start))
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var b strings.Builder
	b.WriteString("# HELP helling_api_requests_total Total HTTP requests served by hellingd.\n")
	b.WriteString("# TYPE helling_api_requests_total counter\n")
	for _, row := range s.metrics.snapshot() {
		labels := metricLabels(row.Key)
		fmt.Fprintf(&b, "helling_api_requests_total{%s} %d\n", labels, row.Sample.Count)
	}

	b.WriteString("# HELP helling_api_errors_total Total HTTP requests returning 5xx errors.\n")
	b.WriteString("# TYPE helling_api_errors_total counter\n")
	for _, row := range s.metrics.snapshot() {
		if row.Sample.Errors == 0 {
			continue
		}
		labels := metricLabels(row.Key)
		fmt.Fprintf(&b, "helling_api_errors_total{%s} %d\n", labels, row.Sample.Errors)
	}

	b.WriteString("# HELP helling_api_request_duration_seconds HTTP request duration histogram.\n")
	b.WriteString("# TYPE helling_api_request_duration_seconds histogram\n")
	for _, row := range s.metrics.snapshot() {
		labels := metricLabels(row.Key)
		for i, upper := range apiDurationBuckets {
			fmt.Fprintf(&b, "helling_api_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, strconv.FormatFloat(upper, 'f', -1, 64), row.Sample.Buckets[i])
		}
		fmt.Fprintf(&b, "helling_api_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, row.Sample.Count)
		fmt.Fprintf(&b, "helling_api_request_duration_seconds_sum{%s} %.9f\n", labels, row.Sample.Sum)
		fmt.Fprintf(&b, "helling_api_request_duration_seconds_count{%s} %d\n", labels, row.Sample.Count)
	}

	b.WriteString("# HELP helling_goroutines Current number of Go goroutines.\n")
	b.WriteString("# TYPE helling_goroutines gauge\n")
	fmt.Fprintf(&b, "helling_goroutines %d\n", runtime.NumGoroutine())

	b.WriteString("# HELP helling_open_connections Current open SQLite connections.\n")
	b.WriteString("# TYPE helling_open_connections gauge\n")
	fmt.Fprintf(&b, "helling_open_connections %d\n", s.cfg.Store.DB().Stats().OpenConnections)

	b.WriteString("# HELP helling_db_size_bytes Current SQLite database size estimated from page_count * page_size.\n")
	b.WriteString("# TYPE helling_db_size_bytes gauge\n")
	if size, err := s.cfg.Store.DBSizeBytes(r.Context()); err == nil {
		fmt.Fprintf(&b, "helling_db_size_bytes %d\n", size)
	} else {
		fmt.Fprintf(&b, "helling_db_size_bytes 0\n")
	}

	b.WriteString("# HELP helling_upstream_metrics_scrape_success Whether the latest upstream metrics scrape succeeded.\n")
	b.WriteString("# TYPE helling_upstream_metrics_scrape_success gauge\n")
	if s.cfg.IncusMetrics == nil {
		b.WriteString("helling_upstream_metrics_scrape_success{upstream=\"incus\"} 0\n")
	} else if upstream, err := s.cfg.IncusMetrics(r.Context()); err == nil {
		b.WriteString("helling_upstream_metrics_scrape_success{upstream=\"incus\"} 1\n")
		if upstream != "" {
			b.WriteString("\n# BEGIN proxied Incus /1.0/metrics\n")
			b.WriteString(upstream)
			b.WriteString("\n# END proxied Incus /1.0/metrics\n")
		}
	} else {
		s.cfg.Logger.Warn("scrape incus metrics", slog.Any("err", err))
		b.WriteString("helling_upstream_metrics_scrape_success{upstream=\"incus\"} 0\n")
	}

	_, _ = w.Write([]byte(b.String()))
}

func metricLabels(key apiMetricKey) string {
	return fmt.Sprintf("method=%q,path=%q,status=%q", escapeMetricLabel(key.Method), escapeMetricLabel(key.Path), escapeMetricLabel(key.Status))
}

func escapeMetricLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
