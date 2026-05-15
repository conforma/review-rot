package github

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
)

const privateKeyEnvVar = "GITHUB_PRIVATE_KEY"

func authenticatedTransport(appID, installationID int64) (http.RoundTripper, error) {
	key := os.Getenv(privateKeyEnvVar)
	if key == "" {
		return nil, fmt.Errorf("%s environment variable is not set", privateKeyEnvVar)
	}
	key = strings.ReplaceAll(key, `\n`, "\n")

	transport, err := ghinstallation.New(http.DefaultTransport, appID, installationID, []byte(key))
	if err != nil {
		return nil, fmt.Errorf("creating GitHub App transport: %w", err)
	}

	return transport, nil
}
