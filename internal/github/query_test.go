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
			CommittedDate     time.Time
			StatusCheckRollup *struct{ State string }
		}
	}{
		{Commit: struct {
			CommittedDate     time.Time
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
			CommittedDate     time.Time
			StatusCheckRollup *struct{ State string }
		}
	}{
		{Commit: struct {
			CommittedDate     time.Time
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

func makeReviewNode(authorType, login, oid string) struct {
	Author struct {
		TypeName string `graphql:"__typename"`
		Login    string
	} `graphql:"author"`
	Commit struct {
		OID string `graphql:"oid"`
	}
} {
	var n struct {
		Author struct {
			TypeName string `graphql:"__typename"`
			Login    string
		} `graphql:"author"`
		Commit struct {
			OID string `graphql:"oid"`
		}
	}
	n.Author.TypeName = authorType
	n.Author.Login = login
	n.Commit.OID = oid
	return n
}

func makeCommentNode(authorType, login string, createdAt time.Time) struct {
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

func setHeadCommitDate(node *prNode, t time.Time) {
	node.Commits.Nodes = []struct {
		Commit struct {
			CommittedDate     time.Time
			StatusCheckRollup *struct{ State string }
		}
	}{
		{Commit: struct {
			CommittedDate     time.Time
			StatusCheckRollup *struct{ State string }
		}{CommittedDate: t}},
	}
}

func TestExtractReviews(t *testing.T) {
	node := prNode{HeadRefOid: "abc123"}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "reviewer", "aaa"),
		makeReviewNode("User", "reviewer", "bbb"),
		makeReviewNode("User", "reviewer", "def456"),
	)

	r := extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (deduplicated by author)", r.Count)
	}
	if !r.HasNewCommits {
		t.Error("HasNewCommits should be true when last review OID differs from head")
	}

	node.Reviews.Nodes[2] = makeReviewNode("User", "reviewer", "abc123")
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
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "reviewer", "aaa"),
		makeReviewNode("Bot", "botuser", "bbb"),
		makeReviewNode("Bot", "botuser", "head"),
	)

	r := extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (bots excluded)", r.Count)
	}
	if !r.HasNewCommits {
		t.Error("HasNewCommits should be true: last human review (aaa) differs from head")
	}

	node.Reviews.Nodes = append(node.Reviews.Nodes[:0],
		makeReviewNode("User", "reviewer", "head"),
		makeReviewNode("Bot", "botuser", "other"),
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
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("Bot", "botuser", "head"),
		makeReviewNode("Bot", "botuser", "old"),
	)

	r := extractReviews(node)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0 (only bot reviews)", r.Count)
	}
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false when there are no human reviews")
	}
}

func TestExtractReviewsExcludesPRAuthor(t *testing.T) {
	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "author", "head"),
		makeReviewNode("User", "author", "old"),
	)

	r := extractReviews(node)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0 (PR author reviews excluded)", r.Count)
	}
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false when there are no peer reviews")
	}
}

func TestExtractReviewsAuthorAndBotsMixed(t *testing.T) {
	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("Bot", "fullsend", "head"),
		makeReviewNode("User", "author", "head"),
		makeReviewNode("Bot", "coderabbit", "old"),
		makeReviewNode("User", "author", "old"),
	)

	r := extractReviews(node)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0 (only bot and PR author reviews)", r.Count)
	}

	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "teammate", "head"),
	)
	r = extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (one peer review)", r.Count)
	}
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false: peer review matches head")
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

func TestExtractReviewsCountsComments(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
	)

	r := extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (one human comment)", r.Count)
	}
}

func TestExtractReviewsCommentsExcludesBots(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
		makeCommentNode("Bot", "codecov", commentDate),
	)

	r := extractReviews(node)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1 (bot comment excluded)", r.Count)
	}
}

func TestExtractReviewsCommentsExcludesAuthor(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "author", commentDate),
	)

	r := extractReviews(node)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0 (PR author comment excluded)", r.Count)
	}
}

func TestExtractReviewsMixedReviewsAndComments(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "reviewer-a", "head"),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer-b", commentDate),
		makeCommentNode("Bot", "ci-bot", commentDate),
		makeCommentNode("User", "author", commentDate),
	)

	r := extractReviews(node)
	if r.Count != 2 {
		t.Errorf("Count = %d, want 2 (1 review + 1 human comment)", r.Count)
	}
}

func TestExtractReviewsCommentAfterHeadCommit(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
	)

	r := extractReviews(node)
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false: comment was posted after HEAD commit")
	}
}

func TestExtractReviewsCommentBeforeHeadCommit(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
	)

	r := extractReviews(node)
	if !r.HasNewCommits {
		t.Error("HasNewCommits should be true: comment was posted before HEAD commit")
	}
}

func TestExtractReviewsCommentCoversHeadDespiteStaleReview(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "reviewer", "old-commit"),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
	)

	r := extractReviews(node)
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false: comment covers HEAD even though review is stale")
	}
}

func TestExtractReviewsReviewCoversHeadDespiteStaleComment(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "reviewer", "head"),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "reviewer", commentDate),
	)

	r := extractReviews(node)
	if r.HasNewCommits {
		t.Error("HasNewCommits should be false: review covers HEAD even though comment is stale")
	}
}

func TestExtractReviewsDeduplicatesByAuthor(t *testing.T) {
	commitDate := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	commentDate := time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

	node := prNode{HeadRefOid: "head"}
	node.Author.Login = "author"
	setHeadCommitDate(&node, commitDate)
	node.Reviews.Nodes = append(node.Reviews.Nodes,
		makeReviewNode("User", "alice", "head"),
		makeReviewNode("User", "alice", "old"),
		makeReviewNode("User", "bob", "head"),
	)
	node.Comments.Nodes = append(node.Comments.Nodes,
		makeCommentNode("User", "alice", commentDate),
		makeCommentNode("User", "bob", commentDate),
		makeCommentNode("User", "charlie", commentDate),
	)

	r := extractReviews(node)
	if r.Count != 3 {
		t.Errorf("Count = %d, want 3 (alice, bob, charlie)", r.Count)
	}
}

func strPtr(s string) *string { return &s }
