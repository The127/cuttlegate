package mcp

import (
	"sync"
	"time"
)

// windowState tracks request count and window expiry for one key.
type windowState struct {
	count     int
	windowEnd time.Time
}

// fixedWindowRateLimiter is a fixed-window, per-key rate limiter.
// It is used to limit evaluate_flag calls per API key.
type fixedWindowRateLimiter struct {
	limit      int
	windowSecs int64

	mu      sync.Mutex
	windows map[string]*windowState
}

// newFixedWindowRateLimiter constructs a fixedWindowRateLimiter.
// limit is the max calls per window; windowSecs is the window duration in seconds.
func newFixedWindowRateLimiter(limit int, windowSecs int64) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		limit:      limit,
		windowSecs: windowSecs,
		windows:    make(map[string]*windowState),
	}
}

// allow returns true if the request for key is within the current window quota.
func (rl *fixedWindowRateLimiter) allow(key string) bool {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	s, ok := rl.windows[key]
	if !ok || now.After(s.windowEnd) {
		rl.windows[key] = &windowState{
			count:     1,
			windowEnd: now.Add(time.Duration(rl.windowSecs) * time.Second),
		}
		return true
	}
	if s.count >= rl.limit {
		return false
	}
	s.count++
	return true
}
