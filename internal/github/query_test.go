package github

import (
	"net/url"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func makeURI(s string) githubv4.URI {
	u, _ := url.Parse(s)
	return githubv4.URI{URL: u}
}

func TestExtractCIStatusSuccess(t *testing.T) {
	node := prNode{}
	node.Commits.Nodes = []struct {
		Commit struct {
			StatusCheckRollup *struct{ State string }
		}
	}{
		{Commit: struct {
			StatusCheckRollup *struct{ State string }
		}{StatusCheckRollup: &struct{ State string }{State: "SUCCESS"}}},
	}

	status := extractCIStatus(node)
	if status == nil || *status != "SUCCESS" {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

func TestExtractCIStatusNull(t *testing.T) {
	node := prNode{}
	node.Commits.Nodes = []struct {
		Commit struct {
			StatusCheckRollup *struct{ State string }
		}
	}{
		{Commit: struct {
			StatusCheckRollup *struct{ State string }
		}{StatusCheckRollup: nil}},
	}

	status := extractCIStatus(node)
	if status != nil {
		t.Errorf("expected nil, got %v", *status)
	}
}

func TestExtractCIStatusNoCommits(t *testing.T) {
	node := prNode{}
	status := extractCIStatus(node)
	if status != nil {
		t.Errorf("expected nil, got %v", *status)
	}
}

func TestExtractSize(t *testing.T) {
	tests := []struct {
		labels []string
		want   *string
	}{
		{[]string{"size: M", "lgtm"}, strPtr("M")},
		{[]string{"lgtm", "approved"}, nil},
		{[]string{"size: XS"}, strPtr("XS")},
		{[]string{"size: XXL", "size: S"}, strPtr("XXL")}, // multiple size labels returns first match
		{nil, nil},
	}

	for _, tt := range tests {
		node := prNode{}
		for _, l := range tt.labels {
			node.Labels.Nodes = append(node.Labels.Nodes, struct{ Name string }{Name: l})
		}
		got := extractSize(node)
		if (got == nil) != (tt.want == nil) {
			t.Errorf("labels=%v: got %v, want %v", tt.labels, got, tt.want)
			continue
		}
		if got != nil && *got != *tt.want {
			t.Errorf("labels=%v: got %q, want %q", tt.labels, *got, *tt.want)
		}
	}
}

func makeReviewNode(authorType, oid string) struct {
	Author struct {
		TypeName string `graphql:"__typename"`
	} `graphql:"author"`
	Commit struct {
		OID string `graphql:"oid"`
	}
} {
	var n struct {
		Author struct {
			TypeName string `graphql:"__typename"`
		} `graphql:"author"`
		Commit struct {
			OID string `graphql:"oid"`
		}
	}
	n.Author.TypeName = authorType
	n.Commit.OID = oid
	return n
}

func TestExtractReviews(t *testing.T) {
	node := prNode{HeadRefOid: "abc123"}
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "aaa"),
		makeReviewNode("User", "bbb"),
		makeReviewNode("User", "def456"),
	)

	r := extractReviews(node)
	if r.Count != 3 {
		t.Errorf("Count = %d, want 3", r.Count)
	}
	if !r.HasNewCommits {
		t.Error("HasNewCommits should be true when last review OID differs from head")
	}

	node.Reviews.Nodes[2] = makeReviewNode("User", "abc123")
	r = extractReviews(node)
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false when last review OID matches head")
	}
}

func TestExtractReviewsZero(t *testing.T) {
	node := prNode{}
	r := extractReviews(node)
	if r.Count != 0 || r.HasNewCommits {
		t.Errorf("expected zero reviews, got count=%d has_new=%v", r.Count, r.HasNewCommits)
	}
}

func TestExtractReviewsExcludesBots(t *testing.T) {
	node := prNode{HeadRefOid: "head"}
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "aaa"),
		makeReviewNode("Bot", "bbb"),
		makeReviewNode("Bot", "head"),
	)

	r := extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (bots excluded)", r.Count)
	}
	if !r.HasNewCommits {
		t.Error("HasNewCommits should be true: last human review (aaa) differs from head")
	}

	node.Reviews.Nodes = append(node.Reviews.Nodes[:0],
		makeReviewNode("User", "head"),
		makeReviewNode("Bot", "other"),
	)
	r = extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1", r.Count)
	}
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false: last human review matches head")
	}
}

func TestExtractReviewsOnlyBots(t *testing.T) {
	node := prNode{HeadRefOid: "head"}
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("Bot", "head"),
		makeReviewNode("Bot", "old"),
	)

	r := extractReviews(node)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0 (only bot reviews)", r.Count)
	}
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false when there are no human reviews")
	}
}

func TestCountUnresolved(t *testing.T) {
	node := prNode{}
	node.ReviewThreads.Nodes = []struct{ IsResolved bool }{
		{IsResolved: true},
		{IsResolved: false},
		{IsResolved: false},
		{IsResolved: true},
	}

	count := countUnresolved(node)
	if count != 2 {
		t.Errorf("expected 2 unresolved, got %d", count)
	}
}

func TestExtractLabels(t *testing.T) {
	node := prNode{}
	node.Labels.Nodes = []struct{ Name string }{
		{Name: "size: M"},
		{Name: "lgtm"},
	}

	labels := extractLabels(node)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if labels[0] != "size: M" || labels[1] != "lgtm" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestTransformPR(t *testing.T) {
	created := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2025, 3, 16, 14, 0, 0, 0, time.UTC)

	node := prNode{
		Title:      "Test PR",
		Number:     42,
		HeadRefOid: "abc",
		IsDraft:    true,
		CreatedAt:  created,
		UpdatedAt:  updated,
	}
	node.URL = makeURI("https://github.com/conforma/policy/pull/42")
	node.Author.TypeName = "User"
	node.Author.Login = "simonbaird"
	node.Author.AvatarURL = makeURI("https://avatars.githubusercontent.com/u/123")

	pr := transformPR(node, "conforma/policy")
	if pr.Title != "Test PR" {
		t.Errorf("Title = %q", pr.Title)
	}
	if pr.Repo != "conforma/policy" {
		t.Errorf("Repo = %q", pr.Repo)
	}
	if pr.Author.Login != "simonbaird" {
		t.Errorf("Author.Login = %q", pr.Author.Login)
	}
	if !pr.IsDraft {
		t.Error("IsDraft should be true")
	}
	if pr.IsAutomated {
		t.Error("IsAutomated should be false for User author")
	}
	if pr.Labels == nil {
		t.Error("Labels should be non-nil empty slice")
	}
	if pr.CreatedAt != "2025-03-15T10:30:00Z" {
		t.Errorf("CreatedAt = %q, want 2025-03-15T10:30:00Z", pr.CreatedAt)
	}
	if pr.UpdatedAt != "2025-03-16T14:00:00Z" {
		t.Errorf("UpdatedAt = %q, want 2025-03-16T14:00:00Z", pr.UpdatedAt)
	}
}

func TestTransformPRBotAuthor(t *testing.T) {
	node := prNode{
		Title:     "Update dependency",
		Number:    99,
		CreatedAt: time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC),
	}
	node.URL = makeURI("https://github.com/conforma/policy/pull/99")
	node.Author.TypeName = "Bot"
	node.Author.Login = "renovate"
	node.Author.AvatarURL = makeURI("https://avatars.githubusercontent.com/in/2740")

	pr := transformPR(node, "conforma/policy")
	if !pr.IsAutomated {
		t.Error("IsAutomated should be true for Bot author")
	}
}

func strPtr(s string) *string { return &s }
