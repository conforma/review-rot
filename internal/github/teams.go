package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

// IsTeamRef checks if an author string is a team reference (@org/team)
func IsTeamRef(author string) bool {
	return strings.HasPrefix(author, "@") && strings.Contains(author, "/")
}

// parseTeamRef extracts org and team slug from @org/team format
func parseTeamRef(ref string) (org, teamSlug string, err error) {
	if !strings.HasPrefix(ref, "@") {
		return "", "", fmt.Errorf("team reference must start with @, got: %q", ref)
	}

	ref = strings.TrimPrefix(ref, "@")
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("team reference must be in format @org/team, got: @%s", ref)
	}

	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("org and team cannot be empty in: @%s", ref)
	}

	return parts[0], parts[1], nil
}

type teamMembersQuery struct {
	Organization struct {
		Team *struct {
			Members struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   githubv4.String
				}
				Nodes []struct {
					Login string
				}
			} `graphql:"members(first: 100, after: $cursor)"`
		} `graphql:"team(slug: $teamSlug)"`
	} `graphql:"organization(login: $org)"`
}

// FetchTeamMembers retrieves all member logins for a GitHub team
func FetchTeamMembers(ctx context.Context, client *githubv4.Client, org, teamSlug string) ([]string, error) {
	var members []string
	variables := map[string]interface{}{
		"org":      githubv4.String(org),
		"teamSlug": githubv4.String(teamSlug),
		"cursor":   (*githubv4.String)(nil),
	}

	for {
		var query teamMembersQuery
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("querying team members: %w", err)
		}

		if query.Organization.Team == nil {
			return nil, fmt.Errorf("team %q not found in organization %q - check that the team exists and the GitHub App has access to it", teamSlug, org)
		}

		for _, node := range query.Organization.Team.Members.Nodes {
			members = append(members, node.Login)
		}

		if !query.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Organization.Team.Members.PageInfo.EndCursor)
	}

	return members, nil
}

// ExpandAuthors expands team references (@org/team) to individual member logins,
// while preserving plain usernames as-is. Returns a deduplicated list.
func ExpandAuthors(ctx context.Context, client *githubv4.Client, authors []string) ([]string, error) {
	seen := make(map[string]bool)
	var expanded []string

	for _, author := range authors {
		if IsTeamRef(author) {
			org, teamSlug, err := parseTeamRef(author)
			if err != nil {
				return nil, fmt.Errorf("invalid team reference %q: %w", author, err)
			}

			members, err := FetchTeamMembers(ctx, client, org, teamSlug)
			if err != nil {
				return nil, fmt.Errorf("failed to expand team %q: %w\nPossible causes:\n  - Team does not exist in organization %q\n  - GitHub App lacks 'Organization members: Read' permission\n  - Team is not visible to the GitHub App installation", author, err, org)
			}

			for _, login := range members {
				lowerLogin := strings.ToLower(login)
				if !seen[lowerLogin] {
					seen[lowerLogin] = true
					expanded = append(expanded, login)
				}
			}
		} else {
			lowerAuthor := strings.ToLower(author)
			if !seen[lowerAuthor] {
				seen[lowerAuthor] = true
				expanded = append(expanded, author)
			}
		}
	}

	return expanded, nil
}
