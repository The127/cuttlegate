package cuttlegate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// sseWireEvent is the JSON wire format for a flag.state_changed SSE event.
type sseWireEvent struct {
	Type        string `json:"type"`
	Project     string `json:"project"`
	Environment string `json:"environment"`
	FlagKey     string `json:"flag_key"`
	Enabled     bool   `json:"enabled"`
	OccurredAt  string `json:"occurred_at"`
}

const (
	sseEventType      = "flag.state_changed"
	backoffInitial    = 100 * time.Millisecond
	backoffCap        = 30 * time.Second
	backoffJitter     = 0.10 // ±10%
)

// Subscribe opens a real-time stream of flag state changes for the given key.
// It returns an update channel, an error channel, and an error if the client
// configuration is invalid. Both channels are closed when ctx is cancelled.
//
// The update channel delivers FlagUpdate values when the flag state changes.
// The error channel delivers non-fatal errors (e.g. reconnect attempts,
// malformed events). A terminal AuthError is sent to the error channel before
// both channels are closed.
//
// Multiple independent calls to Subscribe on the same key return independent
// streams; cancelling one context does not affect the others.
func (c *client) Subscribe(ctx context.Context, key string) (<-chan FlagUpdate, <-chan error, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("cuttlegate: Subscribe: key must not be empty")
	}

	updates := make(chan FlagUpdate, 16)
	errs := make(chan error, 16)

	go c.runSubscribeLoop(ctx, key, updates, errs)

	return updates, errs, nil
}

func (c *client) runSubscribeLoop(ctx context.Context, key string, updates chan<- FlagUpdate, errs chan<- error) {
	defer close(updates)
	defer close(errs)

	url := fmt.Sprintf(
		"%s/api/v1/projects/%s/environments/%s/flags/stream",
		c.baseURL, c.project, c.environment,
	)

	httpClient := c.streamHTTPClient
	attempt := 0

	for {
		// Wait for backoff before retry (not on first attempt).
		if attempt > 0 {
			delay := backoffDelay(attempt)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		result := c.attemptStream(ctx, httpClient, url, key, updates, errs)
		if result == streamResultReconnect {
			attempt++
		} else {
			return // streamResultClosed or streamResultTerminal
		}
	}
}

type streamResult int

const (
	streamResultReconnect streamResult = iota
	streamResultTerminal
	streamResultClosed
)

func (c *client) attemptStream(ctx context.Context, httpClient *http.Client, url, key string, updates chan<- FlagUpdate, errs chan<- error) streamResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// Context already done or bad URL — stop.
		return streamResultClosed
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return streamResultClosed
		}
		// Network error — reconnect.
		sendErr(errs, fmt.Errorf("cuttlegate: SSE connection error: %w", err))
		return streamResultReconnect
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		sendErr(errs, &AuthError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)})
		return streamResultTerminal
	case resp.StatusCode >= 500:
		sendErr(errs, &ServerError{StatusCode: resp.StatusCode, Message: http.StatusText(resp.StatusCode)})
		return streamResultReconnect
	case resp.StatusCode != http.StatusOK:
		sendErr(errs, fmt.Errorf("cuttlegate: SSE unexpected status %d", resp.StatusCode))
		return streamResultReconnect
	}

	// Stream the response body line by line.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return streamResultClosed
		}

		line := scanner.Text()

		// SSE comments (keep-alive) and blank lines — skip.
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
			sendErr(errs, fmt.Errorf("cuttlegate: SSE invalid JSON: %w", err))
			continue
		}

		if wire.Type != sseEventType {
			continue
		}

		// Client-side key filtering.
		if wire.FlagKey != key {
			continue
		}

		updatedAt, err := time.Parse(time.RFC3339, wire.OccurredAt)
		if err != nil {
			sendErr(errs, fmt.Errorf("cuttlegate: SSE invalid occurred_at %q: %w", wire.OccurredAt, err))
			// Non-fatal: still deliver the update with zero time.
			updatedAt = time.Time{}
		}

		select {
		case <-ctx.Done():
			return streamResultClosed
		case updates <- FlagUpdate{
			Key:       wire.FlagKey,
			Enabled:   wire.Enabled,
			UpdatedAt: updatedAt,
		}:
		}
	}

	if ctx.Err() != nil {
		return streamResultClosed
	}

	// Scanner exhausted (server closed the stream) — reconnect.
	return streamResultReconnect
}

// sendErr sends err to ch without blocking. If the channel buffer is full,
// the error is dropped (non-fatal path only; callers using this for terminal
// errors must size the buffer appropriately or read promptly).
func sendErr(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}

// backoffDelay returns the wait duration for the given attempt number.
// Implements exponential backoff with ±10% jitter, capped at backoffCap.
// attempt=1 → ~100ms, attempt=2 → ~200ms, attempt=3 → ~400ms, …, capped at 30s.
func backoffDelay(attempt int) time.Duration {
	// Shift left by (attempt-1) without overflow: clamp to cap first.
	base := backoffInitial << (attempt - 1)
	if base > backoffCap || base <= 0 { // overflow check: base<=0 means shift wrapped
		base = backoffCap
	}
	// ±10% jitter
	jitter := float64(base) * backoffJitter * (2*rand.Float64() - 1)
	result := time.Duration(float64(base) + jitter)
	if result < 0 {
		result = 0
	}
	return result
}
