package auth

import (
	"sync"
	"time"
)

// RateLimiter is a sliding-window per-key counter. Used to throttle login
// attempts. Two keyspaces are expected per docs/standards/security.md:
//
//   - per-username: 5 failures per 15 minutes
//   - per-source-IP: 20 failures per 15 minutes
//
// State is in-memory; restart wipes counters. Persistence deferred to v0.2.
type RateLimiter struct {
	limit  int
	window time.Duration
	mu     sync.Mutex
	hits   map[string][]time.Time
}

// NewRateLimiter returns a limiter that allows up to `limit` events per
// `window` per key.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records an attempt and returns true if it is within the limit.
// Caller should typically only count failures, not successes.
func (r *RateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-r.window)

	r.mu.Lock()
	defer r.mu.Unlock()

	stamps := r.hits[key]
	pruned := stamps[:0]
	for _, t := range stamps {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	if len(pruned) >= r.limit {
		r.hits[key] = pruned
		return false
	}
	pruned = append(pruned, now)
	r.hits[key] = pruned
	return true
}

// Reset clears the counter for a key. Call on successful login.
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hits, key)
}
