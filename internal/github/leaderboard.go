package github

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/conforma/review-rot/internal/model"
	"github.com/shurcooL/githubv4"
)

// reviewerActivityQuery fetches pull requests ordered by most recently updated,
// regardless of state, so recent review activity on merged/closed PRs is counted.
type reviewerActivityQuery struct {
	Repository struct {
		PullRequests struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []reviewerActivityNode
		} `graphql:"pullRequests(first: 50, orderBy: {field: UPDATED_AT, direction: DESC}, after: $cursor)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
	RateLimit struct {
		Cost      int
		Remaining int
		ResetAt   time.Time
	}
}

type reviewerActivityNode struct {
	Title     string
	URL       githubv4.URI
	Number    int
	UpdatedAt time.Time

	Author struct {
		Login string
	} `graphql:"author"`

	Reviews struct {
		Nodes []struct {
			Author struct {
				TypeName string `graphql:"__typename"`
				Login    string
			} `graphql:"author"`
			SubmittedAt time.Time
		}
	} `graphql:"reviews(last: 100, states: [APPROVED, CHANGES_REQUESTED, COMMENTED])"`

	Comments struct {
		Nodes []struct {
			Author struct {
				TypeName string `graphql:"__typename"`
				Login    string
			} `graphql:"author"`
			CreatedAt time.Time
		}
	} `graphql:"comments(last: 100)"`
}

// FetchRepoReviewerActivity walks a repo's pull requests from most-recently
// updated backwards, stopping once it passes the `since` cutoff, and appends
// each reviewer's per-PR engagement into byReviewer (login -> reviewed PRs).
func FetchRepoReviewerActivity(ctx context.Context, client *githubv4.Client, repoFullName string, since time.Time, byReviewer map[string][]model.ReviewedPR) error {
	parts := strings.SplitN(repoFullName, "/", 2)
	if len(parts) != 2 {
		log.Printf("Warning: invalid repo name %q, skipping", repoFullName)
		return nil
	}
	owner, name := parts[0], parts[1]

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(name),
		"cursor": (*githubv4.String)(nil),
	}

	for {
		var query reviewerActivityQuery
		if err := client.Query(ctx, &query, variables); err != nil {
			return err
		}

		log.Printf("  %s: fetched %d PRs (rate limit: %d remaining, resets %s)",
			repoFullName, len(query.Repository.PullRequests.Nodes),
			query.RateLimit.Remaining, query.RateLimit.ResetAt.Format(time.RFC3339))

		reachedCutoff := false
		for _, node := range query.Repository.PullRequests.Nodes {
			// Nodes are ordered by UpdatedAt descending; once we pass the
			// window, every remaining PR is older, so stop paginating.
			if node.UpdatedAt.Before(since) {
				reachedCutoff = true
				break
			}
			for login, engagedAt := range extractPRReviewers(node, since) {
				byReviewer[login] = append(byReviewer[login], model.ReviewedPR{
					Title:     node.Title,
					URL:       node.URL.String(),
					Repo:      repoFullName,
					Number:    node.Number,
					Author:    node.Author.Login,
					EngagedAt: engagedAt.UTC().Format(time.RFC3339),
				})
			}
		}

		if reachedCutoff || !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Repository.PullRequests.PageInfo.EndCursor)
	}

	return nil
}

// extractPRReviewers maps each distinct login that reviewed or commented on the
// PR within the window to the time of their most recent such activity. It
// excludes bots and the PR author, mirroring the reviewer-counting rules in
// extractReviews, with an added per-event window check.
func extractPRReviewers(node reviewerActivityNode, since time.Time) map[string]time.Time {
	engaged := make(map[string]time.Time)
	author := node.Author.Login

	consider := func(login, typeName string, ts time.Time) {
		if isBotLogin(login, typeName) || login == "" || login == author {
			return
		}
		if ts.Before(since) {
			return
		}
		if cur, ok := engaged[login]; !ok || ts.After(cur) {
			engaged[login] = ts
		}
	}

	for _, review := range node.Reviews.Nodes {
		consider(review.Author.Login, review.Author.TypeName, review.SubmittedAt)
	}
	for _, comment := range node.Comments.Nodes {
		consider(comment.Author.Login, comment.Author.TypeName, comment.CreatedAt)
	}

	return engaged
}

// BuildLeaderboard turns aggregated per-reviewer PRs into a sorted leaderboard,
// ranked by review count descending then login ascending. Each reviewer's PRs
// are sorted most-recent engagement first.
func BuildLeaderboard(byReviewer map[string][]model.ReviewedPR, windowDays int, since time.Time) *model.Leaderboard {
	reviewers := make([]model.ReviewerStat, 0, len(byReviewer))
	for login, prs := range byReviewer {
		// EngagedAt is RFC3339 in UTC, so lexical sort equals chronological.
		sort.Slice(prs, func(i, j int) bool {
			return prs[i].EngagedAt > prs[j].EngagedAt
		})
		reviewers = append(reviewers, model.ReviewerStat{
			Login:   login,
			Reviews: len(prs),
			PRs:     prs,
		})
	}
	sort.Slice(reviewers, func(i, j int) bool {
		if reviewers[i].Reviews != reviewers[j].Reviews {
			return reviewers[i].Reviews > reviewers[j].Reviews
		}
		return reviewers[i].Login < reviewers[j].Login
	})

	return &model.Leaderboard{
		WindowDays: windowDays,
		Since:      since.UTC().Format(time.RFC3339),
		Reviewers:  reviewers,
	}
}
