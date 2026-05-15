package github

import (
	"context"
	"log"

	"github.com/shurcooL/githubv4"
)

func DiscoverOrgRepos(ctx context.Context, client *githubv4.Client, org string) ([]string, error) {
	var query struct {
		Organization struct {
			Repositories struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   githubv4.String
				}
				Nodes []struct {
					NameWithOwner string
					IsArchived    bool
				}
			} `graphql:"repositories(first: 100, after: $cursor)"`
		} `graphql:"organization(login: $org)"`
	}

	var repos []string
	variables := map[string]interface{}{
		"org":    githubv4.String(org),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		for _, repo := range query.Organization.Repositories.Nodes {
			if !repo.IsArchived {
				repos = append(repos, repo.NameWithOwner)
			}
		}

		if !query.Organization.Repositories.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Organization.Repositories.PageInfo.EndCursor)
	}

	return repos, nil
}

func CollectRepos(ctx context.Context, client *githubv4.Client, orgs []string, explicitRepos []string) []string {
	var discovered []string

	for _, org := range orgs {
		repos, err := DiscoverOrgRepos(ctx, client, org)
		if err != nil {
			log.Printf("Warning: failed to discover repos for org %s: %v", org, err)
			continue
		}
		discovered = append(discovered, repos...)
	}

	return collectReposFromLists(discovered, explicitRepos)
}

func collectReposFromLists(discovered, explicit []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, repo := range discovered {
		if !seen[repo] {
			seen[repo] = true
			result = append(result, repo)
		}
	}

	for _, repo := range explicit {
		if !seen[repo] {
			seen[repo] = true
			result = append(result, repo)
		}
	}

	return result
}
