package github

import (
	"net/http"
	"time"

	"github.com/shurcooL/githubv4"
)

const (
	httpTimeout = 30 * time.Second
	maxRetries  = 2 // Initial attempt + 2 retries = 3 total attempts
)

func NewClient(appID, installationID int64) (*githubv4.Client, error) {
	transport, err := authenticatedTransport(appID, installationID)
	if err != nil {
		return nil, err
	}

	// Wrap authenticated transport with retry logic for 5xx errors
	retryTransport := newRetryRoundTripper(transport, maxRetries)

	httpClient := &http.Client{
		Transport: retryTransport,
		Timeout:   httpTimeout,
	}
	return githubv4.NewClient(httpClient), nil
}
