package github

import (
	"testing"
)

func TestAuthenticatedTransportMissingKey(t *testing.T) {
	t.Setenv(privateKeyEnvVar, "")

	_, err := authenticatedTransport(245286, 59973090)
	if err == nil {
		t.Fatal("expected error for missing private key")
	}
}

func TestAuthenticatedTransportInvalidKey(t *testing.T) {
	t.Setenv(privateKeyEnvVar, "not-a-valid-pem-key")

	_, err := authenticatedTransport(245286, 59973090)
	if err == nil {
		t.Fatal("expected error for invalid private key")
	}
}
