package github

import (
	"net/http"
	"time"

	"github.com/shurcooL/githubv4"
)

func NewClient(appID, installationID int64) (*githubv4.Client, error) {
	transport, err := authenticatedTransport(appID, installationID)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	return githubv4.NewClient(httpClient), nil
}
