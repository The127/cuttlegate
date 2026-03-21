package httpadapter

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/karo/cuttlegate/internal/domain"
)

// windowState tracks request count and window start for a single key.
type windowState struct {
	count     int
	windowEnd time.Time
}

// RateLimiter is a fixed-window, per-user-ID rate limiter for HTTP handlers.
type RateLimiter struct {
	limit      int
	windowSize time.Duration

	mu      sync.Mutex
	windows map[string]*windowState
}

// NewRateLimiter constructs a RateLimiter that allows at most limit requests
// per user ID within each windowSize period.
func NewRateLimiter(limit int, windowSize time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:      limit,
		windowSize: windowSize,
		windows:    make(map[string]*windowState),
	}
}

// Limit returns middleware that enforces the rate limit. Requests are keyed by
// the user ID in the AuthContext. If no AuthContext is present the request is
// rejected with 403 — callers must place auth middleware before this one.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := domain.AuthContextFrom(r.Context())
		if !ok || ac.UserID == "" {
			WriteError(w, domain.ErrForbidden)
			return
		}

		if !rl.allow(ac.UserID) {
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.windowSize.Seconds())))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow returns true if the request for key is within the current window quota.
func (rl *RateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	s, ok := rl.windows[key]
	if !ok || now.After(s.windowEnd) {
		rl.windows[key] = &windowState{count: 1, windowEnd: now.Add(rl.windowSize)}
		return true
	}

	if s.count >= rl.limit {
		return false
	}
	s.count++
	return true
}
