package github

import (
	"testing"
	"time"

	"github.com/conforma/review-rot/internal/model"
)

func makeActivityReview(authorType, login string, submittedAt time.Time) struct {
	Author struct {
		TypeName string `graphql:"__typename"`
		Login    string
	} `graphql:"author"`
	SubmittedAt time.Time
} {
	var n struct {
		Author struct {
			TypeName string `graphql:"__typename"`
			Login    string
		} `graphql:"author"`
		SubmittedAt time.Time
	}
	n.Author.TypeName = authorType
	n.Author.Login = login
	n.SubmittedAt = submittedAt
	return n
}

func makeActivityComment(authorType, login string, createdAt time.Time) struct {
	Author struct {
		TypeName string `graphql:"__typename"`
		Login    string
	} `graphql:"author"`
	CreatedAt time.Time
} {
	var n struct {
		Author struct {
			TypeName string `graphql:"__typename"`
			Login    string
		} `graphql:"author"`
		CreatedAt time.Time
	}
	n.Author.TypeName = authorType
	n.Author.Login = login
	n.CreatedAt = createdAt
	return n
}

func TestExtractPRReviewersBasic(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	inWindow := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)

	node := reviewerActivityNode{}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeActivityReview("User", "alice", inWindow),
		makeActivityReview("User", "bob", inWindow),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeActivityComment("User", "charlie", inWindow),
	)

	got := extractPRReviewers(node, since)
	want := map[string]bool{"alice": true, "bob": true, "charlie": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for l := range want {
		if _, ok := got[l]; !ok {
			t.Errorf("missing reviewer %q in %v", l, got)
		}
	}
}

func TestExtractPRReviewersExcludesBotsSelfAndEmpty(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	inWindow := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)

	node := reviewerActivityNode{}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeActivityReview("Bot", "renovate", inWindow),
		makeActivityReview("User", "konflux-ci-qe-bot", inWindow), // machine user, bot login suffix
		makeActivityReview("User", "author", inWindow),            // self-review
		makeActivityReview("User", "", inWindow),                  // ghost/deleted user
		makeActivityReview("User", "alice", inWindow),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeActivityComment("Bot", "codecov", inWindow),
		makeActivityComment("User", "author", inWindow),
	)

	got := extractPRReviewers(node, since)
	if len(got) != 1 {
		t.Fatalf("got %v, want only alice", got)
	}
	if _, ok := got["alice"]; !ok {
		t.Errorf("got %v, want alice", got)
	}
}

func TestExtractPRReviewersWindowFiltering(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	after := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)

	node := reviewerActivityNode{}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeActivityReview("User", "old-reviewer", before), // outside window
		makeActivityReview("User", "new-reviewer", after),  // inside window
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeActivityComment("User", "old-commenter", before), // outside window
	)

	got := extractPRReviewers(node, since)
	if len(got) != 1 {
		t.Fatalf("got %v, want only new-reviewer", got)
	}
	if _, ok := got["new-reviewer"]; !ok {
		t.Errorf("got %v, want new-reviewer", got)
	}
}

func TestExtractPRReviewersUsesLatestEngagement(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	later := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	node := reviewerActivityNode{}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeActivityReview("User", "alice", earlier),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeActivityComment("User", "alice", later),
	)

	got := extractPRReviewers(node, since)
	if len(got) != 1 {
		t.Fatalf("got %v, want single alice entry", got)
	}
	if !got["alice"].Equal(later) {
		t.Errorf("alice engagement = %v, want latest %v", got["alice"], later)
	}
}

func TestBuildLeaderboardSortsAndTieBreaks(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	byReviewer := map[string][]model.ReviewedPR{
		"alice":   prRefs(5),
		"bob":     prRefs(10),
		"charlie": prRefs(5),
		"dave":    prRefs(1),
	}

	lb := BuildLeaderboard(byReviewer, 30, since, nil)

	if lb.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30", lb.WindowDays)
	}
	if lb.Since != "2025-03-01T00:00:00Z" {
		t.Errorf("Since = %q, want 2025-03-01T00:00:00Z", lb.Since)
	}

	wantOrder := []string{"bob", "alice", "charlie", "dave"}
	if len(lb.Reviewers) != len(wantOrder) {
		t.Fatalf("got %d reviewers, want %d", len(lb.Reviewers), len(wantOrder))
	}
	for i, login := range wantOrder {
		if lb.Reviewers[i].Login != login {
			t.Errorf("Reviewers[%d].Login = %q, want %q (order: %+v)", i, lb.Reviewers[i].Login, login, lb.Reviewers)
		}
	}
	if lb.Reviewers[0].Reviews != 10 {
		t.Errorf("top reviewer count = %d, want 10", lb.Reviewers[0].Reviews)
	}
	if len(lb.Reviewers[0].PRs) != 10 {
		t.Errorf("top reviewer PRs = %d, want 10", len(lb.Reviewers[0].PRs))
	}
}

func TestBuildLeaderboardSortsPRsByRecency(t *testing.T) {
	byReviewer := map[string][]model.ReviewedPR{
		"alice": {
			{Repo: "o/r", Number: 1, EngagedAt: "2025-03-05T00:00:00Z"},
			{Repo: "o/r", Number: 2, EngagedAt: "2025-03-20T00:00:00Z"},
			{Repo: "o/r", Number: 3, EngagedAt: "2025-03-10T00:00:00Z"},
		},
	}

	lb := BuildLeaderboard(byReviewer, 30, time.Now(), nil)
	prs := lb.Reviewers[0].PRs
	wantNumbers := []int{2, 3, 1} // most recent engagement first
	for i, n := range wantNumbers {
		if prs[i].Number != n {
			t.Errorf("PRs[%d].Number = %d, want %d (order: %+v)", i, prs[i].Number, n, prs)
		}
	}
}

func TestBuildLeaderboardEmpty(t *testing.T) {
	lb := BuildLeaderboard(map[string][]model.ReviewedPR{}, 14, time.Now(), nil)
	if lb == nil {
		t.Fatal("expected non-nil leaderboard")
	}
	if lb.Reviewers == nil {
		t.Error("expected non-nil (empty) reviewers slice")
	}
	if len(lb.Reviewers) != 0 {
		t.Errorf("expected 0 reviewers, got %d", len(lb.Reviewers))
	}
	if lb.WindowDays != 14 {
		t.Errorf("WindowDays = %d, want 14", lb.WindowDays)
	}
}

func TestBuildLeaderboardFiltersToTeam(t *testing.T) {
	since := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	byReviewer := map[string][]model.ReviewedPR{
		"alice":     prRefs(3),
		"outsider":  prRefs(5),
		"BohdanMar": prRefs(2),
	}

	// Allowlist uses different casing to confirm the match is case-insensitive.
	lb := BuildLeaderboard(byReviewer, 30, since, []string{"Alice", "bohdanmar"})

	if len(lb.Reviewers) != 2 {
		t.Fatalf("got %d reviewers, want 2 (team only): %+v", len(lb.Reviewers), lb.Reviewers)
	}
	seen := map[string]bool{}
	for _, r := range lb.Reviewers {
		seen[r.Login] = true
	}
	if !seen["alice"] || !seen["BohdanMar"] {
		t.Errorf("expected alice and BohdanMar, got %+v", lb.Reviewers)
	}
	if seen["outsider"] {
		t.Errorf("outsider should be excluded, got %+v", lb.Reviewers)
	}
}

// prRefs builds n placeholder ReviewedPR entries for count-based assertions.
func prRefs(n int) []model.ReviewedPR {
	prs := make([]model.ReviewedPR, n)
	for i := range prs {
		prs[i] = model.ReviewedPR{Repo: "o/r", Number: i + 1, EngagedAt: "2025-03-10T00:00:00Z"}
	}
	return prs
}
