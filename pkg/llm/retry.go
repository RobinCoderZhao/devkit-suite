package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"
)

// retryClient wraps any Client with exponential backoff retry logic.
//
// Backoff schedule (baseDelay=1s):
//
//	attempt 0 → 1s  (±25% jitter → 0.75s-1.25s)
//	attempt 1 → 2s  (±25% jitter → 1.5s-2.5s)
//	attempt 2 → 4s  (±25% jitter → 3.0s-5.0s)
//	attempt 3 → 8s  (±25% jitter → 6.0s-10.0s)
//	attempt 4 → 16s (±25% jitter → 12.0s-20.0s)
//	capped at 60s
type retryClient struct {
	inner      Client
	maxRetries int
	baseDelay  time.Duration
}

// wrapWithRetry wraps a client with exponential backoff retry logic.
func wrapWithRetry(client Client, maxRetries int) Client {
	if maxRetries <= 1 {
		return client
	}
	return &retryClient{
		inner:      client,
		maxRetries: maxRetries,
		baseDelay:  1 * time.Second, // 1s base for 1s, 2s, 4s, 8s... progression
	}
}

func (r *retryClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		resp, err := r.inner.Generate(ctx, req)
		if err == nil {
			if attempt > 0 {
				slog.Info("LLM request succeeded after retry", "attempt", attempt+1)
			}
			return resp, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return nil, err
		}

		delay := r.backoffDelay(attempt)
		slog.Warn("LLM request failed, retrying with exponential backoff",
			"attempt", attempt+1,
			"max_retries", r.maxRetries,
			"backoff", delay.Round(time.Millisecond),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, fmt.Errorf("max retries (%d) exceeded: %w", r.maxRetries, lastErr)
}

func (r *retryClient) GenerateJSON(ctx context.Context, req *Request, out any) error {
	req.JSONMode = true
	resp, err := r.Generate(ctx, req)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(resp.Content), out); err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}
	return nil
}

func (r *retryClient) Provider() Provider {
	return r.inner.Provider()
}

func (r *retryClient) Close() error {
	return r.inner.Close()
}

// backoffDelay calculates exponential backoff with ±25% jitter.
// Formula: baseDelay * 2^attempt * (0.75 + rand(0, 0.5))
func (r *retryClient) backoffDelay(attempt int) time.Duration {
	base := float64(r.baseDelay) * math.Pow(2, float64(attempt))

	// Add ±25% jitter to prevent thundering herd
	jitter := 0.75 + rand.Float64()*0.5 // range [0.75, 1.25]
	delay := time.Duration(base * jitter)

	// Cap at 60 seconds
	const maxDelay = 60 * time.Second
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

// isRetryableError determines if an error is worth retrying.
// Retries on: 429 (rate limit), 500/502/503 (server errors), timeouts, connection resets.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	for _, keyword := range []string{"429", "500", "502", "503", "timeout", "connection reset", "EOF", "high demand"} {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	return false
}
