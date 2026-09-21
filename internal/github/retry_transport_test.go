package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockRoundTripper allows simulating HTTP responses for testing
type mockRoundTripper struct {
	responses []*http.Response
	errors    []error
	callCount int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.callCount >= len(m.responses) {
		if m.callCount < len(m.errors) && m.errors[m.callCount] != nil {
			err := m.errors[m.callCount]
			m.callCount++
			return nil, err
		}
		return nil, nil
	}
	resp := m.responses[m.callCount]
	var err error
	if m.callCount < len(m.errors) {
		err = m.errors[m.callCount]
	}
	m.callCount++
	return resp, err
}

func TestRetryOn503(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 200, Body: io.NopCloser(strings.NewReader("success"))},
		},
		errors: []error{nil, nil, nil},
	}

	rt := newRetryRoundTripper(mock, 2)
	req, _ := http.NewRequest("GET", "https://api.github.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", mock.callCount)
	}
}

func TestNoRetryOn404(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			{StatusCode: 404, Body: io.NopCloser(strings.NewReader("not found"))},
		},
		errors: []error{nil},
	}

	rt := newRetryRoundTripper(mock, 2)
	req, _ := http.NewRequest("GET", "https://api.github.com", nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	if mock.callCount != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", mock.callCount)
	}
}

func TestExhaustRetries(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			{StatusCode: 502, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 502, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 502, Body: io.NopCloser(strings.NewReader(""))},
		},
		errors: []error{nil, nil, nil},
	}

	rt := newRetryRoundTripper(mock, 2)
	req, _ := http.NewRequest("GET", "https://api.github.com", nil)

	resp, err := rt.RoundTrip(req)
	// Should return the last 502 response after exhausting retries
	if resp == nil || resp.StatusCode != 502 {
		t.Errorf("expected 502 response after exhausting retries, got status: %v, err: %v", resp, err)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 attempts (1 initial + 2 retries), got %d", mock.callCount)
	}
}

func TestRespectsContextCancellation(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))},
		},
		errors: []error{nil, nil},
	}

	rt := newRetryRoundTripper(mock, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com", nil)

	start := time.Now()
	_, err := rt.RoundTrip(req)
	duration := time.Since(start)

	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
	if duration > 2*time.Second {
		t.Errorf("retry should have been cancelled quickly, took %v", duration)
	}
}

func TestRetryDifferent5xxCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"500 Internal Server Error", 500},
		{"502 Bad Gateway", 502},
		{"503 Service Unavailable", 503},
		{"504 Gateway Timeout", 504},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRoundTripper{
				responses: []*http.Response{
					{StatusCode: tc.statusCode, Body: io.NopCloser(strings.NewReader(""))},
					{StatusCode: 200, Body: io.NopCloser(strings.NewReader("success"))},
				},
				errors: []error{nil, nil},
			}

			rt := newRetryRoundTripper(mock, 2)
			req, _ := http.NewRequest("GET", "https://api.github.com", nil)

			resp, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("expected success after retry, got error: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
			}
			if mock.callCount != 2 {
				t.Errorf("expected 2 attempts (1 initial + 1 retry), got %d", mock.callCount)
			}
		})
	}
}

func TestRequestBodyPreservation(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			{StatusCode: 503, Body: io.NopCloser(strings.NewReader(""))},
			{StatusCode: 200, Body: io.NopCloser(strings.NewReader("success"))},
		},
		errors: []error{nil, nil},
	}

	rt := newRetryRoundTripper(mock, 2)
	body := strings.NewReader(`{"query": "test"}`)
	req, _ := http.NewRequest("POST", "https://api.github.com/graphql", body)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if mock.callCount != 2 {
		t.Errorf("expected 2 attempts, got %d", mock.callCount)
	}
}
