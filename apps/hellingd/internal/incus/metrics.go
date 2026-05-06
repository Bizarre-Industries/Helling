package incus

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ScrapeMetrics fetches Incus' native Prometheus exposition from /1.0/metrics.
func ScrapeMetrics(ctx context.Context, socketPath string) (string, error) {
	if socketPath == "" {
		socketPath = "/var/lib/incus/unix.socket.user"
	}
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := &http.Client{Transport: UnixTransport(socketPath), Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(scrapeCtx, http.MethodGet, "http://incus/1.0/metrics", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build Incus metrics request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("scrape Incus metrics: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("scrape Incus metrics: upstream returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read Incus metrics: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}
