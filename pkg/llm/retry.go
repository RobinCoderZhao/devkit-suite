package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// retryClient wraps any Client with retry logic.
type retryClient struct {
	inner      Client
	maxRetries int
	baseDelay  time.Duration
}

// wrapWithRetry wraps a client with retry logic.
func wrapWithRetry(client Client, maxRetries int) Client {
	if maxRetries <= 1 {
		return client
	}
	return &retryClient{
		inner:      client,
		maxRetries: maxRetries,
		baseDelay:  500 * time.Millisecond,
	}
}

func (r *retryClient) Generate(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		resp, err := r.inner.Generate(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return nil, err
		}

		delay := r.backoffDelay(attempt)
		slog.Warn("LLM request failed, retrying",
			"attempt", attempt+1,
			"max_retries", r.maxRetries,
			"delay", delay,
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

func (r *retryClient) backoffDelay(attempt int) time.Duration {
	delay := float64(r.baseDelay) * math.Pow(2, float64(attempt))
	maxDelay := 30 * time.Second
	if time.Duration(delay) > maxDelay {
		return maxDelay
	}
	return time.Duration(delay)
}

// isRetryableError determines if an error is worth retrying.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Retry on rate limits, server errors, and timeouts
	for _, keyword := range []string{"429", "500", "502", "503", "timeout", "connection reset"} {
		if contains(errStr, keyword) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
