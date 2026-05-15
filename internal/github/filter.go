package github

import (
	"strings"

	"github.com/conforma/review-rot/internal/model"
)

func FilterPRs(prs []model.PullRequest, coreOrgs []string, teamAuthors []string) []model.PullRequest {
	orgSet := make(map[string]bool, len(coreOrgs))
	for _, org := range coreOrgs {
		orgSet[strings.ToLower(org)] = true
	}

	authorSet := make(map[string]bool, len(teamAuthors))
	for _, author := range teamAuthors {
		authorSet[strings.ToLower(author)] = true
	}

	var filtered []model.PullRequest
	for _, pr := range prs {
		repoOrg := strings.ToLower(repoOwner(pr.Repo))
		if orgSet[repoOrg] {
			filtered = append(filtered, pr)
			continue
		}
		if authorSet[strings.ToLower(pr.Author.Login)] {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func MarkBots(prs []model.PullRequest, bots []string) {
	botSet := make(map[string]bool, len(bots))
	for _, bot := range bots {
		botSet[strings.ToLower(bot)] = true
	}

	for i := range prs {
		if botSet[strings.ToLower(prs[i].Author.Login)] {
			prs[i].IsAutomated = true
		}
	}
}

func repoOwner(repo string) string {
	parts := strings.SplitN(repo, "/", 2)
	return parts[0]
}
