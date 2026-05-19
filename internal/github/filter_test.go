package github

import (
	"testing"

	"github.com/conforma/review-rot/internal/model"
)

func makePR(repo, author string) model.PullRequest {
	return model.PullRequest{
		Repo:   repo,
		Author: model.Author{Login: author},
		Labels: []string{},
	}
}

func TestFilterPRsCoreOrg(t *testing.T) {
	prs := []model.PullRequest{
		makePR("conforma/policy", "outsider"),
		makePR("enterprise-contract/ec-cli", "outsider"),
		makePR("konflux-ci/build-definitions", "outsider"),
	}

	filtered := FilterPRs(prs, []string{"conforma", "enterprise-contract"}, []string{"teamuser"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs from core orgs, got %d", len(filtered))
	}
}

func TestFilterPRsTeamAuthor(t *testing.T) {
	prs := []model.PullRequest{
		makePR("konflux-ci/build-definitions", "simonbaird"),
		makePR("konflux-ci/build-definitions", "outsider"),
	}

	filtered := FilterPRs(prs, []string{"conforma"}, []string{"simonbaird"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 PR from team author, got %d", len(filtered))
	}
	if filtered[0].Author.Login != "simonbaird" {
		t.Errorf("expected simonbaird, got %s", filtered[0].Author.Login)
	}
}

func TestFilterPRsCaseInsensitive(t *testing.T) {
	prs := []model.PullRequest{
		makePR("Conforma/policy", "user"),
		makePR("konflux-ci/x", "SimonBaird"),
	}

	filtered := FilterPRs(prs, []string{"conforma"}, []string{"simonbaird"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs (case-insensitive match), got %d", len(filtered))
	}
}

func TestFilterPRsNilInputs(t *testing.T) {
	result := FilterPRs(nil, nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 PRs, got %d", len(result))
	}

	prs := []model.PullRequest{makePR("org/repo", "user")}
	result = FilterPRs(prs, nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 PRs with no orgs or authors, got %d", len(result))
	}
}

