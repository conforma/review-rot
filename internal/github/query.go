package github

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/conforma/review-rot/internal/model"
	"github.com/shurcooL/githubv4"
)

type prQueryResult struct {
	Repository struct {
		PullRequests struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []prNode
		} `graphql:"pullRequests(first: 100, states: OPEN, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
	RateLimit struct {
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
}

type prNode struct {
	Title      string
	URL        githubv4.URI
	Number     int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	IsDraft    bool
	HeadRefOid string

	Author struct {
		TypeName  string       `graphql:"__typename"`
		Login     string
		AvatarURL githubv4.URI `graphql:"avatarUrl"`
	} `graphql:"author"`

	Labels struct {
		Nodes []struct {
			Name string
		}
	} `graphql:"labels(first: 20)"`

	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string
				}
			}
		}
	} `graphql:"commits(last: 1)"`

	Reviews struct {
		TotalCount int
		Nodes      []struct {
			Commit struct {
				OID string `graphql:"oid"`
			}
		}
	} `graphql:"reviews(last: 1, states: [APPROVED, CHANGES_REQUESTED, COMMENTED])"`

	ReviewThreads struct {
		Nodes []struct {
			IsResolved bool
		}
	} `graphql:"reviewThreads(first: 100)"`
}

func FetchRepoPRs(ctx context.Context, client *githubv4.Client, repoFullName string) ([]model.PullRequest, error) {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		log.Printf("Warning: invalid repo name %q, skipping", repoFullName)
		return nil, nil
	}
	owner, name := parts[0], parts[1]

	var allPRs []model.PullRequest
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		var query prQueryResult
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		log.Printf("  %s: fetched %d PRs (rate limit: %d remaining, resets %s)",
			repoFullName, len(query.Repository.PullRequests.Nodes),
			query.RateLimit.Remaining, query.RateLimit.ResetAt.Format(time.RFC3339))

		for _, node := range query.Repository.PullRequests.Nodes {
			allPRs = append(allPRs, transformPR(node, repoFullName))
		}

		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Repository.PullRequests.PageInfo.EndCursor)
	}

	return allPRs, nil
}

func transformPR(node prNode, repo string) model.PullRequest {
	pr := model.PullRequest{
		Title:     node.Title,
		URL:       node.URL.String(),
		Number:    node.Number,
		Repo:      repo,
		CreatedAt: node.CreatedAt.Format(time.RFC3339),
		UpdatedAt: node.UpdatedAt.Format(time.RFC3339),
		IsDraft:   node.IsDraft,
		Author: model.Author{
			Login:     node.Author.Login,
			AvatarURL: node.Author.AvatarURL.String(),
		},
		IsAutomated: node.Author.TypeName == "Bot",
	}

	pr.CIStatus = extractCIStatus(node)
	pr.Size = extractSize(node)
	pr.Reviews = extractReviews(node)
	pr.UnresolvedConversations = countUnresolved(node)
	pr.Labels = extractLabels(node)

	return pr
}

func extractCIStatus(node prNode) *string {
	if len(node.Commits.Nodes) == 0 {
		return nil
	}
	commit := node.Commits.Nodes[0].Commit
	if commit.StatusCheckRollup == nil {
		return nil
	}
	s := commit.StatusCheckRollup.State
	return &s
}

func extractSize(node prNode) *string {
	for _, label := range node.Labels.Nodes {
		if strings.HasPrefix(label.Name, "size: ") {
			size := strings.TrimPrefix(label.Name, "size: ")
			return &size
		}
	}
	return nil
}

func extractReviews(node prNode) model.Reviews {
	r := model.Reviews{
		Count: node.Reviews.TotalCount,
	}
	if r.Count > 0 && len(node.Reviews.Nodes) > 0 {
		lastReviewOID := node.Reviews.Nodes[0].Commit.OID
		r.HasNewCommits = lastReviewOID != node.HeadRefOid
	}
	return r
}

func countUnresolved(node prNode) int {
	count := 0
	for _, thread := range node.ReviewThreads.Nodes {
		if !thread.IsResolved {
			count++
		}
	}
	return count
}

func extractLabels(node prNode) []string {
	labels := make([]string, 0, len(node.Labels.Nodes))
	for _, l := range node.Labels.Nodes {
		labels = append(labels, l.Name)
	}
	return labels
}
