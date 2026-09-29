package github

import (
	"encoding/json"
	"net/url"
	"strconv"
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

func makeReviewNodeState(authorType, login, oid, state string) struct {
	ID     string
	Author struct {
		TypeName string `graphql:"__typename"`
		Login    string
	} `graphql:"author"`
	Commit struct {
		OID string `graphql:"oid"`
	}
	State string
} {
	var n struct {
		ID     string
		Author struct {
			TypeName string `graphql:"__typename"`
			Login    string
		} `graphql:"author"`
		Commit struct {
			OID string `graphql:"oid"`
		}
		State string
	}
	n.Author.TypeName = authorType
	n.Author.Login = login
	n.Commit.OID = oid
	n.State = state
	return n
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

func TestTransformPRReviewsJSONContract(t *testing.T) {
	node := reviewPRNode()
	addReviewEvents(&node, []reviewEvent{
		{"User", "alice", "head", "APPROVED"},
		{"User", "bob", "head", "CHANGES_REQUESTED"},
		{"User", "carol", "old", "CHANGES_REQUESTED"},
	})
	pr := transformPR(node, "conforma/policy")
	if pr.Reviews.ApprovedCount != 1 || !pr.Reviews.OutstandingChangeRequestsOnHead || pr.UnresolvedConversations != 2 {
		t.Errorf("PR = %+v, want one head approval, a head request and two outstanding requests", pr)
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Reviews json.RawMessage `json:"reviews"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	want := `{"approved_count":1,"outstanding_change_requests_on_head":true}`
	if string(output.Reviews) != want {
		t.Errorf("reviews JSON = %s, want %s", output.Reviews, want)
	}
}

type reviewEvent struct {
	authorType, login, oid, state string
}

func addReviewEvents(node *prNode, events []reviewEvent) {
	for _, event := range events {
		review := makeReviewNodeState(event.authorType, event.login, event.oid, event.state)
		review.ID = strconv.Itoa(len(node.Reviews.Nodes) + 1)
		node.Reviews.Nodes = append(node.Reviews.Nodes, review)
	}
}

func reviewPRNode() prNode {
	node := prNode{HeadRefOid: "head"}
	node.URL = makeURI("https://github.com/conforma/policy/pull/1")
	node.Author.Login = "author"
	node.Author.AvatarURL = makeURI("https://avatars.githubusercontent.com/u/123")
	return node
}

func TestTransformPRCurrentHeadReviews(t *testing.T) {
	tests := []struct {
		name          string
		events        []reviewEvent
		wantApprovals int
	}{
		{"no decisions", nil, 0},
		{"stale approval", []reviewEvent{{"User", "alice", "old", "APPROVED"}}, 0},
		{"distinct approvals despite repeated and neutral reviews", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "APPROVED"},
			{"User", "bob", "head", "APPROVED"},
			{"User", "bob", "old", "COMMENTED"},
		}, 2},
		{"later current-head request supersedes head approval", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "CHANGES_REQUESTED"},
		}, 0},
		{"later approval on an old SHA does not revoke head approval", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "old", "APPROVED"},
		}, 1},
		{"change request does not add approval", []reviewEvent{{"User", "alice", "head", "CHANGES_REQUESTED"}}, 0},
		{"commented review does not add approval", []reviewEvent{
			{"User", "alice", "old", "APPROVED"},
			{"User", "alice", "head", "COMMENTED"},
		}, 0},
		{"dismissed review does not add approval", []reviewEvent{{"User", "alice", "head", "DISMISSED"}}, 0},
		{"dismissed later review does not revoke earlier approval", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "DISMISSED"},
		}, 1},
		{"bots, author and unknown accounts are not reviewers", []reviewEvent{
			{"Bot", "app", "head", "APPROVED"},
			{"User", "machine-bot", "head", "APPROVED"},
			{"User", "author", "head", "APPROVED"},
			{"User", "", "head", "APPROVED"},
			{"Mannequin", "unknown", "head", "APPROVED"},
		}, 0},
		{"human reviewer alongside ineligible accounts", []reviewEvent{
			{"Bot", "app", "head", "APPROVED"},
			{"User", "alice", "head", "APPROVED"},
			{"User", "author", "head", "APPROVED"},
		}, 1},
		{"GitHub login casing cannot double-count or self-approve", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "Alice", "head", "APPROVED"},
			{"User", "AUTHOR", "head", "APPROVED"},
		}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := reviewPRNode()
			addReviewEvents(&node, tt.events)
			reviews := transformPR(node, "conforma/policy").Reviews
			if reviews.ApprovedCount != tt.wantApprovals {
				t.Errorf("reviews = %+v, want approvals=%d", reviews, tt.wantApprovals)
			}
		})
	}
}

func TestTransformPROutstandingChangeRequests(t *testing.T) {
	tests := []struct {
		name           string
		events         []reviewEvent
		threads        []bool
		wantUnresolved int
		wantOnHead     bool
		wantApprovals  int
	}{
		{"current request", []reviewEvent{{"User", "alice", "head", "CHANGES_REQUESTED"}}, nil, 1, true, 0},
		{"old request", []reviewEvent{{"User", "alice", "old", "CHANGES_REQUESTED"}}, nil, 1, false, 0},
		{"ineligible requests do not count", []reviewEvent{
			{"Bot", "app", "head", "CHANGES_REQUESTED"},
			{"User", "machine-bot", "head", "CHANGES_REQUESTED"},
			{"User", "author", "head", "CHANGES_REQUESTED"},
			{"User", "", "head", "CHANGES_REQUESTED"},
			{"Mannequin", "unknown", "head", "CHANGES_REQUESTED"},
		}, []bool{false}, 1, false, 0},
		{"each outstanding review plus unresolved threads", []reviewEvent{
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "bob", "head", "CHANGES_REQUESTED"},
		}, []bool{false, false, true}, 5, true, 0},
		{"commented review and resolved thread do not clear request", []reviewEvent{
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "COMMENTED"},
		}, []bool{true}, 1, false, 0},
		{"another reviewer approving does not clear request", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "bob", "head", "APPROVED"},
		}, nil, 1, true, 1},
		{"approval on old SHA clears same reviewer request", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "old", "APPROVED"},
		}, nil, 0, false, 0},
		{"old-SHA approval clears request but cannot restore superseded head approval", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "old", "APPROVED"},
		}, nil, 0, false, 0},
		{"approval on head clears old request", []reviewEvent{
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "APPROVED"},
		}, nil, 0, false, 1},
		{"approval clears same reviewer request across login casing", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "ALICE", "old", "APPROVED"},
		}, nil, 0, false, 0},
		{"current-head request after approval supersedes it and remains outstanding", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "CHANGES_REQUESTED"},
		}, nil, 1, true, 0},
		{"old-SHA request after head approval leaves approval counted", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "old", "CHANGES_REQUESTED"},
		}, nil, 1, false, 1},
		{"dismissal only affects dismissed review", []reviewEvent{
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "DISMISSED"},
		}, nil, 1, false, 0},
		{"dismissed request alone", []reviewEvent{{"User", "alice", "head", "DISMISSED"}}, nil, 0, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := reviewPRNode()
			addReviewEvents(&node, tt.events)
			for _, resolved := range tt.threads {
				node.ReviewThreads.Nodes = append(node.ReviewThreads.Nodes, struct{ IsResolved bool }{resolved})
			}
			pr := transformPR(node, "conforma/policy")
			if pr.UnresolvedConversations != tt.wantUnresolved || pr.Reviews.ApprovedCount != tt.wantApprovals {
				t.Errorf("PR = %+v, want unresolved=%d approvals=%d", pr, tt.wantUnresolved, tt.wantApprovals)
			}

			data, err := json.Marshal(pr)
			if err != nil {
				t.Fatal(err)
			}
			var output struct {
				Reviews map[string]json.RawMessage `json:"reviews"`
			}
			if err := json.Unmarshal(data, &output); err != nil {
				t.Fatal(err)
			}
			value, exists := output.Reviews["outstanding_change_requests_on_head"]
			if !exists {
				t.Fatal("reviews.outstanding_change_requests_on_head missing from JSON")
			}
			var onHead bool
			if err := json.Unmarshal(value, &onHead); err != nil {
				t.Fatal(err)
			}
			if onHead != tt.wantOnHead {
				t.Errorf("outstanding_change_requests_on_head = %t, want %t", onHead, tt.wantOnHead)
			}
		})
	}
}

func TestTransformPRDismissedReviewID(t *testing.T) {
	for _, tt := range []struct {
		name            string
		events          []reviewEvent
		dismissedReview int
		wantUnresolved  int
		wantOnHead      bool
		wantApprovals   int
	}{
		{"dismissed head request no longer counts", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "DISMISSED"},
		}, 0, 0, false, 0},
		{"dismissal leaves another old request by same reviewer", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "DISMISSED"},
		}, 0, 1, false, 0},
		{"dismissing old request leaves another current-head request", []reviewEvent{
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "old", "CHANGES_REQUESTED"},
			{"User", "alice", "old", "DISMISSED"},
		}, 1, 1, true, 0},
		{"dismissing same-head request restores earlier approval", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "CHANGES_REQUESTED"},
			{"User", "alice", "head", "DISMISSED"},
		}, 1, 0, false, 1},
		{"dismissed head approval no longer counts", []reviewEvent{
			{"User", "alice", "head", "APPROVED"},
			{"User", "alice", "head", "DISMISSED"},
		}, 0, 0, false, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			node := reviewPRNode()
			addReviewEvents(&node, tt.events)
			node.Reviews.Nodes[len(node.Reviews.Nodes)-1].ID = node.Reviews.Nodes[tt.dismissedReview].ID
			pr := transformPR(node, "conforma/policy")
			if pr.UnresolvedConversations != tt.wantUnresolved ||
				pr.Reviews.OutstandingChangeRequestsOnHead != tt.wantOnHead ||
				pr.Reviews.ApprovedCount != tt.wantApprovals {
				t.Errorf("PR = %+v, want unresolved=%d on_head=%t approvals=%d",
					pr, tt.wantUnresolved, tt.wantOnHead, tt.wantApprovals)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestIsBotLogin(t *testing.T) {
	cases := []struct {
		login    string
		typeName string
		want     bool
	}{
		{"coderabbitai", "Bot", true},       // GitHub App bot
		{"konflux-ci-qe-bot", "User", true}, // machine user, -bot suffix
		{"dependabot[bot]", "User", true},   // [bot] suffix without Bot typename
		{"Konflux-CI-QE-Bot", "User", true}, // suffix match is case-insensitive
		{"alice", "User", false},            // human
		{"", "User", false},                 // ghost/deleted user
		{"robotics-fan", "User", false},     // "bot" mid-word, not a suffix
	}
	for _, c := range cases {
		if got := isBotLogin(c.login, c.typeName); got != c.want {
			t.Errorf("isBotLogin(%q, %q) = %v, want %v", c.login, c.typeName, got, c.want)
		}
	}
}
