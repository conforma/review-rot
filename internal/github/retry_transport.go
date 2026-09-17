package github

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// retryRoundTripper wraps an http.RoundTripper and retries requests
// that fail with 5xx HTTP status codes using exponential backoff.
type retryRoundTripper struct {
	base       http.RoundTripper
	maxRetries int
}

// newRetryRoundTripper creates a retry-capable transport wrapper.
// maxRetries controls the maximum number of retry attempts (not including initial request).
func newRetryRoundTripper(base http.RoundTripper, maxRetries int) *retryRoundTripper {
	return &retryRoundTripper{
		base:       base,
		maxRetries: maxRetries,
	}
}

// RoundTrip implements http.RoundTripper with retry logic for 5xx errors.
func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request body for potential retries
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var lastResp *http.Response
	var lastErr error
	var lastRespBody []byte // Buffer for terminal 5xx response body

	operation := func() error {
		// Restore body for retry attempts
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := r.base.RoundTrip(req)
		lastResp = resp
		lastErr = err

		// Network/transport errors are not retried (e.g., DNS, connection refused)
		if err != nil {
			return backoff.Permanent(err)
		}

		// Only retry on 5xx server errors
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			// Buffer response body before closing so it can be restored if retries exhaust
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if readErr == nil {
				lastRespBody = body
			}

			return fmt.Errorf("server error: HTTP %d", resp.StatusCode)
		}

		// Success or non-retryable error (4xx, etc.)
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 30 * time.Second
	b.MaxElapsedTime = 0 // No time limit, only retry count
	b.Multiplier = 2.0
	b.RandomizationFactor = 0.1 // Add 10% jitter

	retryBackoff := backoff.WithMaxRetries(b, uint64(r.maxRetries))
	retryBackoff = backoff.WithContext(retryBackoff, req.Context())

	notifyFunc := func(err error, duration time.Duration) {
		log.Printf("GitHub API request failed (%v), retrying in %s", err, duration)
	}

	err := backoff.RetryNotify(operation, retryBackoff, notifyFunc)

	// Check if context was cancelled
	if req.Context().Err() != nil {
		return lastResp, req.Context().Err()
	}

	if err != nil {
		// If we exhausted retries on 5xx, return the last response with restored body
		if lastResp != nil && lastResp.StatusCode >= 500 {
			if lastRespBody != nil {
				lastResp.Body = io.NopCloser(bytes.NewReader(lastRespBody))
			}
			return lastResp, nil
		}
		return lastResp, lastErr
	}

	return lastResp, nil
}
