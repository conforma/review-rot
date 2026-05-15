package model

import "time"

type Output struct {
	GeneratedAt  time.Time     `json:"generated_at"`
	UISettings   *UISettings   `json:"ui_settings,omitempty"`
	PullRequests []PullRequest `json:"pull_requests"`
}

type UISettings struct {
	Title   string     `json:"title,omitempty"`
	Logo    string     `json:"logo,omitempty"`
	Favicon string     `json:"favicon,omitempty"`
	Palette *UIPalette `json:"palette,omitempty"`
}

type UIPalette struct {
	Accent      string `json:"accent,omitempty"`
	AccentDark  string `json:"accent_dark,omitempty"`
	AccentLight string `json:"accent_light,omitempty"`
}

func NewUISettings(title, logo, favicon, accent, accentDark, accentLight string) *UISettings {
	if title == "" && logo == "" && favicon == "" && accent == "" && accentDark == "" && accentLight == "" {
		return nil
	}
	s := &UISettings{
		Title:   title,
		Logo:    logo,
		Favicon: favicon,
	}
	if accent != "" || accentDark != "" || accentLight != "" {
		s.Palette = &UIPalette{
			Accent:      accent,
			AccentDark:  accentDark,
			AccentLight: accentLight,
		}
	}
	return s
}

type PullRequest struct {
	Title                   string   `json:"title"`
	URL                     string   `json:"url"`
	Number                  int      `json:"number"`
	Repo                    string   `json:"repo"`
	Author                  Author   `json:"author"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
	IsDraft                 bool     `json:"is_draft"`
	IsAutomated             bool     `json:"is_automated"`
	CIStatus                *string  `json:"ci_status"`
	Size                    *string  `json:"size"`
	Reviews                 Reviews  `json:"reviews"`
	UnresolvedConversations int      `json:"unresolved_conversations"`
	Labels                  []string `json:"labels"`
}

type Author struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}

type Reviews struct {
	Count         int  `json:"count"`
	HasNewCommits bool `json:"has_new_commits"`
}
