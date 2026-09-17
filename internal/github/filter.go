package github

import (
	"strings"

	"github.com/conforma/review-rot/internal/model"
)

func FilterPRs(prs []model.PullRequest, coreOrgs []string, watchAllRepos []string, teamAuthors []string) []model.PullRequest {
	orgSet := make(map[string]bool, len(coreOrgs))
	for _, org := range coreOrgs {
		orgSet[strings.ToLower(org)] = true
	}

	watchAllRepoSet := make(map[string]bool, len(watchAllRepos))
	for _, repo := range watchAllRepos {
		watchAllRepoSet[strings.ToLower(repo)] = true
	}

	authorSet := make(map[string]bool, len(teamAuthors))
	for _, author := range teamAuthors {
		authorSet[strings.ToLower(author)] = true
	}

	var filtered []model.PullRequest
	for _, pr := range prs {
		repoOrg := strings.ToLower(repoOwner(pr.Repo))
		fullRepo := strings.ToLower(pr.Repo)

		// Priority 1: Core org
		if orgSet[repoOrg] {
			filtered = append(filtered, pr)
			continue
		}

		// Priority 2: Watch-all repo
		if watchAllRepoSet[fullRepo] {
			filtered = append(filtered, pr)
			continue
		}

		// Priority 3: Team author
		if authorSet[strings.ToLower(pr.Author.Login)] {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func repoOwner(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	return parts[0]
}
