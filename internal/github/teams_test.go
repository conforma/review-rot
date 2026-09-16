package github

import (
	"testing"
)

func TestIsTeamRef(t *testing.T) {
	tests := []struct {
		author string
		want   bool
	}{
		{"@myorg/backend-team", true},
		{"@conforma/developers", true},
		{"@org/team-name-with-dashes", true},
		{"simonbaird", false},
		{"@invalid", false},
		{"", false},
		{"org/team", false},
	}

	for _, tt := range tests {
		t.Run(tt.author, func(t *testing.T) {
			got := IsTeamRef(tt.author)
			if got != tt.want {
				t.Errorf("IsTeamRef(%q) = %v, want %v", tt.author, got, tt.want)
			}
		})
	}
}

func TestParseTeamRef(t *testing.T) {
	tests := []struct {
		ref         string
		wantOrg     string
		wantTeam    string
		shouldError bool
	}{
		{"@myorg/backend-team", "myorg", "backend-team", false},
		{"@conforma/developers", "conforma", "developers", false},
		{"@org/team-name-with-dashes", "org", "team-name-with-dashes", false},
		{"simonbaird", "", "", true},
		{"@invalid", "", "", true},
		{"@/team", "", "", true},
		{"@org/", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			org, team, err := parseTeamRef(tt.ref)
			if tt.shouldError {
				if err == nil {
					t.Errorf("parseTeamRef(%q) should error, got org=%q, team=%q", tt.ref, org, team)
				}
			} else {
				if err != nil {
					t.Errorf("parseTeamRef(%q) unexpected error: %v", tt.ref, err)
				}
				if org != tt.wantOrg {
					t.Errorf("parseTeamRef(%q) org = %q, want %q", tt.ref, org, tt.wantOrg)
				}
				if team != tt.wantTeam {
					t.Errorf("parseTeamRef(%q) team = %q, want %q", tt.ref, team, tt.wantTeam)
				}
			}
		})
	}
}
