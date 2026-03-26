package cuttlegate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CachedClient wraps a Client and maintains a local flag state map. It seeds
// the cache via EvaluateAll on Bootstrap and keeps it fresh via a single
// background SSE connection. Bool and String read from cache; they fall back
// to a live HTTP call on cache miss.
//
// CachedClient satisfies the Client interface — callers can substitute it for
// a plain Client transparently.
//
// Usage:
//
//	cc, err := cuttlegate.NewCachedClient(cfg)
//	if err != nil { ... }
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	if err := cc.Bootstrap(ctx, evalCtx); err != nil { ... }
//	enabled, err := cc.Bool(ctx, "dark-mode", evalCtx)
type CachedClient struct {
	inner  Client
	cfg    Config
	mu     sync.RWMutex
	cache  map[string]EvalResult
	stopFn context.CancelFunc // stops the background SSE goroutine
}

// compile-time assertion: *CachedClient must satisfy Client.
var _ Client = (*CachedClient)(nil)

// NewCachedClient constructs a CachedClient from the given Config.
// Returns an error if any required Config field is missing. No network calls
// are made — call Bootstrap to seed the cache.
func NewCachedClient(cfg Config) (*CachedClient, error) {
	inner, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Store == nil {
		cfg.Store = NoopFlagStore{}
	}
	return &CachedClient{
		inner: inner,
		cfg:   cfg,
		cache: make(map[string]EvalResult),
	}, nil
}

// Bootstrap seeds the cache by calling EvaluateAll, then starts a single
// background SSE goroutine that applies flag.state_changed events to the
// cache. Returns an error if EvaluateAll fails.
//
// ctx controls the lifetime of the background SSE goroutine — cancel it to
// stop the goroutine cleanly. Bootstrap may be called more than once; the
// previous goroutine is stopped and the cache is replaced with a fresh
// EvaluateAll result.
func (cc *CachedClient) Bootstrap(ctx context.Context, ec EvalContext) error {
	results, evalErr := cc.inner.EvaluateAll(ctx, ec)
	if evalErr != nil {
		// EvaluateAll failed — try loading from the store as fallback.
		loaded, loadErr := cc.cfg.Store.Load(ctx)
		if loadErr != nil || len(loaded) == 0 {
			return evalErr
		}
		results = loaded
	} else {
		// Persist the fresh results (fire-and-forget).
		_ = cc.cfg.Store.Save(ctx, results)
	}

	// Derive a child context for the SSE goroutine so we can stop it without
	// cancelling the caller's ctx.
	sseCtx, sseCancel := context.WithCancel(ctx)

	cc.mu.Lock()
	if cc.stopFn != nil {
		cc.stopFn()
	}
	cc.cache = results
	cc.stopFn = sseCancel
	cc.mu.Unlock()

	go cc.runCacheSSELoop(sseCtx)
	return nil
}

// Bool returns the boolean value of the flag identified by key for the given
// EvalContext. It reads from the cache; on a cache miss it falls back to a
// live HTTP evaluation using ec. The fallback result is not written to the
// cache.
//
// Returns false and NotFoundError if the flag is not found.
func (cc *CachedClient) Bool(ctx context.Context, key string, ec EvalContext) (bool, error) {
	cc.mu.RLock()
	result, ok := cc.cache[key]
	cc.mu.RUnlock()

	if ok {
		return result.Variant == "true", nil
	}
	return cc.inner.Bool(ctx, key, ec)
}

// String returns the string value of the flag identified by key for the given
// EvalContext. It reads from the cache; on a cache miss it falls back to a
// live HTTP evaluation using ec. The fallback result is not written to the
// cache.
//
// Returns "" and NotFoundError if the flag is not found.
func (cc *CachedClient) String(ctx context.Context, key string, ec EvalContext) (string, error) {
	cc.mu.RLock()
	result, ok := cc.cache[key]
	cc.mu.RUnlock()

	if ok {
		return result.Variant, nil
	}
	return cc.inner.String(ctx, key, ec)
}

// EvaluateAll delegates to the inner client. It does not use the cache.
func (cc *CachedClient) EvaluateAll(ctx context.Context, ec EvalContext) (map[string]EvalResult, error) {
	return cc.inner.EvaluateAll(ctx, ec)
}

// Evaluate delegates to the inner client. It does not use the cache.
func (cc *CachedClient) Evaluate(ctx context.Context, key string, ec EvalContext) (EvalResult, error) {
	return cc.inner.Evaluate(ctx, key, ec)
}

// EvaluateFlag delegates to the inner client. It does not use the cache.
func (cc *CachedClient) EvaluateFlag(ctx context.Context, key string, ec EvalContext) (FlagResult, error) {
	return cc.inner.EvaluateFlag(ctx, key, ec)
}

// Subscribe delegates to the inner client. The returned stream is independent
// of the cache SSE connection maintained by Bootstrap.
func (cc *CachedClient) Subscribe(ctx context.Context, key string) (<-chan FlagUpdate, <-chan error, error) {
	return cc.inner.Subscribe(ctx, key)
}

// runCacheSSELoop connects to the /flags/stream endpoint and applies
// flag.state_changed events to the cache. It only updates keys that are
// already present in the cache (keys from Bootstrap); unknown keys are
// ignored. On 401/403 the goroutine terminates without reconnect. On 5xx or
// network error it reconnects with exponential backoff. It exits cleanly when
// ctx is cancelled.
func (cc *CachedClient) runCacheSSELoop(ctx context.Context) {
	// Extract the fields we need from the inner client's config. We keep a
	// copy of cfg so we don't need to type-assert the Client interface.
	baseURL := cc.cfg.BaseURL
	project := cc.cfg.Project
	environment := cc.cfg.Environment
	token := cc.cfg.ServiceToken

	// Build a stream HTTP client (no timeout — SSE connections are long-lived).
	streamClient := cc.cfg.StreamHTTPClient
	if streamClient == nil {
		streamClient = &http.Client{}
	}

	url := fmt.Sprintf(
		"%s/api/v1/projects/%s/environments/%s/flags/stream",
		baseURL, project, environment,
	)

	attempt := 0
	for {
		if attempt > 0 {
			delay := backoffDelay(attempt)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		result := cc.attemptCacheStream(ctx, streamClient, url, token)
		if result == streamResultReconnect {
			attempt++
		} else {
			return // streamResultClosed or streamResultTerminal
		}
	}
}

func (cc *CachedClient) attemptCacheStream(ctx context.Context, httpClient *http.Client, url, token string) streamResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return streamResultClosed
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return streamResultClosed
		}
		return streamResultReconnect
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return streamResultTerminal
	case resp.StatusCode >= 500:
		return streamResultReconnect
	case resp.StatusCode != http.StatusOK:
		return streamResultReconnect
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return streamResultClosed
		}

		line := scanner.Text()
		if strings.HasPrefix(line, ":") || line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if jsonStr == "" {
			continue
		}

		var wire sseWireEvent
		if err := json.Unmarshal([]byte(jsonStr), &wire); err != nil {
			continue
		}
		if wire.Type != sseEventType {
			continue
		}

		cc.applyCacheUpdate(wire)
		_ = cc.cfg.Store.Save(ctx, cc.cacheSnapshot())
	}

	if ctx.Err() != nil {
		return streamResultClosed
	}
	return streamResultReconnect
}

// cacheSnapshot returns a shallow copy of the current cache under the read lock.
func (cc *CachedClient) cacheSnapshot() map[string]EvalResult {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	snapshot := make(map[string]EvalResult, len(cc.cache))
	for k, v := range cc.cache {
		snapshot[k] = v
	}
	return snapshot
}

// applyCacheUpdate writes the SSE event into the cache only if the flag key
// is already present. Unknown keys (not seeded by Bootstrap) are ignored.
func (cc *CachedClient) applyCacheUpdate(wire sseWireEvent) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	existing, ok := cc.cache[wire.FlagKey]
	if !ok {
		return
	}

	variant := "false"
	if wire.Enabled {
		variant = "true"
	}
	existing.Enabled = wire.Enabled
	existing.Variant = variant
	cc.cache[wire.FlagKey] = existing
}
