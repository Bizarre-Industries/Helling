package auth

import (
	"errors"
	"sync"
	"time"
)

// ErrRateLimited is returned by Login / LoginWithMFA when the (username, ip)
// pair has exceeded the failed-login threshold within the sliding window.
// API layer maps this to HTTP 429 (huma.Error429TooManyRequests).
var ErrRateLimited = errors.New("auth: too many failed login attempts")

// Sliding-window failed-login limiter.
//
// Defaults align with the alpha-gate Auth checklist:
//
//	Rate limiting: 6 failed logins → 429
//
// Six failed attempts within loginRateWindow trigger a lockout that lasts
// until the oldest counted attempt slides out of the window. The window is
// per (username, IP) so a single shared NAT IP cannot lock out an entire
// username and a single attacker cannot freely cycle usernames.
const (
	loginRateMaxAttempts = 6
	loginRateWindow      = 15 * time.Minute
)

type loginAttemptStore struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	now      func() time.Time // injectable for tests
}

func newLoginAttemptStore() *loginAttemptStore {
	return &loginAttemptStore{
		attempts: make(map[string][]time.Time),
		now:      time.Now,
	}
}

// allow reports whether a fresh attempt by this key is allowed. It does not
// record the attempt; callers record on failure via fail() and clear on
// success via reset().
func (s *loginAttemptStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(key)
	return len(s.attempts[key]) < loginRateMaxAttempts
}

// fail records a failed attempt. Subsequent allow() checks see the new count.
func (s *loginAttemptStore) fail(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gcLocked(key)
	s.attempts[key] = append(s.attempts[key], s.now())
}

// reset clears the attempt record for a key (called after successful login).
func (s *loginAttemptStore) reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, key)
}

// gcLocked drops timestamps older than the sliding window. Caller holds mu.
func (s *loginAttemptStore) gcLocked(key string) {
	cutoff := s.now().Add(-loginRateWindow)
	src := s.attempts[key]
	if len(src) == 0 {
		return
	}
	keep := src[:0]
	for _, t := range src {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		delete(s.attempts, key)
		return
	}
	s.attempts[key] = keep
}

// loginRateKey builds the per-(username, ip) attempt key. Both inputs are
// already normalized by the API layer (lowercased / trimmed).
func loginRateKey(username, ip string) string {
	return username + "|" + ip
}
