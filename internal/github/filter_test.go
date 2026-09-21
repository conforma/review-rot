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

	filtered := FilterPRs(prs, []string{"conforma", "enterprise-contract"}, nil, []string{"teamuser"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs from core orgs, got %d", len(filtered))
	}
}

func TestFilterPRsTeamAuthor(t *testing.T) {
	prs := []model.PullRequest{
		makePR("konflux-ci/build-definitions", "simonbaird"),
		makePR("konflux-ci/build-definitions", "outsider"),
	}

	filtered := FilterPRs(prs, []string{"conforma"}, nil, []string{"simonbaird"})
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

	filtered := FilterPRs(prs, []string{"conforma"}, nil, []string{"simonbaird"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs (case-insensitive match), got %d", len(filtered))
	}
}

func TestFilterPRsNilInputs(t *testing.T) {
	result := FilterPRs(nil, nil, nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 PRs, got %d", len(result))
	}

	prs := []model.PullRequest{makePR("org/repo", "user")}
	result = FilterPRs(prs, nil, nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 PRs with no orgs or authors, got %d", len(result))
	}
}

func TestFilterPRsWatchAllRepo(t *testing.T) {
	prs := []model.PullRequest{
		makePR("konflux-ci/build-definitions", "outsider"),
		makePR("konflux-ci/other-repo", "outsider"),
	}

	// build-definitions is watch-all, other-repo is not
	filtered := FilterPRs(prs, nil, []string{"konflux-ci/build-definitions"}, []string{"teamuser"})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 PR from watch-all repo, got %d", len(filtered))
	}
	if filtered[0].Repo != "konflux-ci/build-definitions" {
		t.Errorf("expected konflux-ci/build-definitions, got %s", filtered[0].Repo)
	}
}

func TestFilterPRsWatchAllVsNormal(t *testing.T) {
	prs := []model.PullRequest{
		makePR("owner/watch-all-repo", "outsider"),
		makePR("owner/normal-repo", "outsider"),
		makePR("owner/normal-repo", "teamuser"),
	}

	filtered := FilterPRs(prs, nil, []string{"owner/watch-all-repo"}, []string{"teamuser"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs (1 from watch-all, 1 from team member), got %d", len(filtered))
	}

	// Check that we got the watch-all PR and the team member PR
	foundWatchAll := false
	foundTeamMember := false
	for _, pr := range filtered {
		if pr.Repo == "owner/watch-all-repo" && pr.Author.Login == "outsider" {
			foundWatchAll = true
		}
		if pr.Repo == "owner/normal-repo" && pr.Author.Login == "teamuser" {
			foundTeamMember = true
		}
	}
	if !foundWatchAll {
		t.Error("expected PR from watch-all repo by outsider")
	}
	if !foundTeamMember {
		t.Error("expected PR from normal repo by team member")
	}
}

func TestFilterPRsWatchAllCaseInsensitive(t *testing.T) {
	prs := []model.PullRequest{
		makePR("Owner/Repo", "outsider"),
	}

	filtered := FilterPRs(prs, nil, []string{"owner/repo"}, nil)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 PR (case-insensitive watch-all), got %d", len(filtered))
	}
}

func TestFilterPRsPriority(t *testing.T) {
	prs := []model.PullRequest{
		makePR("conforma/repo", "outsider"),             // Core org - should be included
		makePR("konflux-ci/build-definitions", "outsider"), // Watch-all - should be included
		makePR("tektoncd/chains", "teamuser"),          // Team member - should be included
		makePR("tektoncd/chains", "outsider"),          // Not core, not watch-all, not team - excluded
	}

	filtered := FilterPRs(
		prs,
		[]string{"conforma"},
		[]string{"konflux-ci/build-definitions"},
		[]string{"teamuser"},
	)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 PRs (org + watch-all + team), got %d", len(filtered))
	}
}

func TestFilterPRsWatchAllEmpty(t *testing.T) {
	prs := []model.PullRequest{
		makePR("conforma/policy", "outsider"),
		makePR("konflux-ci/build-definitions", "teamuser"),
	}

	// Empty watch-all list - should behave like before
	filtered := FilterPRs(prs, []string{"conforma"}, []string{}, []string{"teamuser"})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 PRs (backward compat), got %d", len(filtered))
	}
}

