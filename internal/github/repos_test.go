package github

import "testing"

func TestCollectReposDedup(t *testing.T) {
	discovered := []string{"conforma/policy", "conforma/cli", "conforma/review-rot"}
	explicit := []string{"conforma/policy", "konflux-ci/build-definitions"}

	result := collectReposFromLists(discovered, explicit)

	if len(result) != 4 {
		t.Fatalf("expected 4 repos, got %d: %v", len(result), result)
	}

	expected := map[string]bool{
		"conforma/policy":              true,
		"conforma/cli":                 true,
		"conforma/review-rot":          true,
		"konflux-ci/build-definitions": true,
	}
	seen := make(map[string]bool, len(result))
	for _, r := range result {
		if !expected[r] {
			t.Errorf("unexpected repo: %s", r)
		}
		if seen[r] {
			t.Errorf("duplicate repo in result: %s", r)
		}
		seen[r] = true
	}
	for r := range expected {
		if !seen[r] {
			t.Errorf("missing expected repo: %s", r)
		}
	}
}

func TestCollectReposEmptyInputs(t *testing.T) {
	result := collectReposFromLists(nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(result))
	}

	result = collectReposFromLists(nil, []string{"a/b"})
	if len(result) != 1 || result[0] != "a/b" {
		t.Fatalf("expected [a/b], got %v", result)
	}
}

func TestCollectReposPreservesOrder(t *testing.T) {
	result := collectReposFromLists(
		[]string{"z/repo", "a/repo"},
		[]string{"m/repo"},
	)
	if len(result) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(result))
	}
	if result[0] != "z/repo" || result[1] != "a/repo" || result[2] != "m/repo" {
		t.Errorf("order not preserved: %v", result)
	}
}
