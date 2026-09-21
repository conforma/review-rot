package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	GitHub      GitHubConfig      `yaml:"github"`
	Sources     SourcesConfig     `yaml:"sources"`
	Authors     []string          `yaml:"authors"`
	Leaderboard LeaderboardConfig `yaml:"leaderboard"`
	UI          UIConfig          `yaml:"-"`
}

// defaultLeaderboardWindowDays is the data horizon used when
// leaderboard.window_days is omitted. The frontend interval selector can narrow
// this further, so it is the widest range the leaderboard can show.
const defaultLeaderboardWindowDays = 90

type LeaderboardConfig struct {
	WindowDays int `yaml:"window_days"`
}

type GitHubConfig struct {
	AppID          int64 `yaml:"app_id"`
	InstallationID int64 `yaml:"installation_id"`
}

type SourcesConfig struct {
	Orgs  []OrgConfig `yaml:"orgs"`
	Repos []string    `yaml:"repos"` // YAML input, may contain + prefixes

	// Derived fields populated during validation
	normalRepos   []string // Repos without + (filter by team authors)
	watchAllRepos []string // Repos with + (all authors), prefix stripped
}

type OrgConfig struct {
	Name string `yaml:"name"`
}

type UIConfig struct {
	Title   string    `yaml:"title"`
	Logo    string    `yaml:"logo"`
	Favicon string    `yaml:"favicon"`
	Palette UIPalette `yaml:"palette"`
}

type UIPalette struct {
	Accent      string `yaml:"accent"`
	AccentDark  string `yaml:"accent_dark"`
	AccentLight string `yaml:"accent_light"`
}

// AllRepos returns all repos (both normal and watch-all) for GitHub API queries.
// The + prefix is stripped from watch-all repos.
func (s *SourcesConfig) AllRepos() []string {
	seen := make(map[string]bool)
	var result []string

	for _, repo := range s.watchAllRepos {
		if !seen[repo] {
			seen[repo] = true
			result = append(result, repo)
		}
	}

	for _, repo := range s.normalRepos {
		if !seen[repo] {
			seen[repo] = true
			result = append(result, repo)
		}
	}

	return result
}

// WatchAllRepos returns repos that should include PRs from all authors.
func (s *SourcesConfig) WatchAllRepos() []string {
	return s.watchAllRepos
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return validate(&cfg)
}

func LoadDir(dir string) (*Config, error) {
	cfg, err := Load(filepath.Join(dir, "sources.yaml"))
	if err != nil {
		return nil, err
	}

	uiPath := filepath.Join(dir, "ui.yaml")
	uiData, err := os.ReadFile(uiPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading ui config: %w", err)
	}

	if err := yaml.Unmarshal(uiData, &cfg.UI); err != nil {
		return nil, fmt.Errorf("parsing ui config: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) (*Config, error) {
	if cfg.GitHub.AppID == 0 {
		return nil, fmt.Errorf("config: github.app_id is required")
	}
	if cfg.GitHub.InstallationID == 0 {
		return nil, fmt.Errorf("config: github.installation_id is required")
	}
	if len(cfg.Sources.Orgs) == 0 && len(cfg.Sources.Repos) == 0 {
		return nil, fmt.Errorf("config: at least one org or repo must be configured")
	}
	if cfg.Leaderboard.WindowDays <= 0 {
		cfg.Leaderboard.WindowDays = defaultLeaderboardWindowDays
	}

	// Parse repo entries and separate watch-all (+prefix) from normal repos
	for i, repo := range cfg.Sources.Repos {
		// Trim whitespace first to handle quoted entries with leading/trailing spaces
		repo = strings.TrimSpace(repo)
		watchAll := strings.HasPrefix(repo, "+")
		cleanRepo := strings.TrimPrefix(repo, "+")

		// Validate repo format
		if cleanRepo == "" {
			return nil, fmt.Errorf("config: repos[%d]: empty repo name", i)
		}

		parts := strings.Split(cleanRepo, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("config: repos[%d]: invalid format %q (expected owner/repo)", i, repo)
		}

		// Categorize repo
		if watchAll {
			cfg.Sources.watchAllRepos = append(cfg.Sources.watchAllRepos, cleanRepo)
		} else {
			cfg.Sources.normalRepos = append(cfg.Sources.normalRepos, cleanRepo)
		}
	}

	return cfg, nil
}

func (c *Config) OrgNames() []string {
	names := make([]string, len(c.Sources.Orgs))
	for i, org := range c.Sources.Orgs {
		names[i] = org.Name
	}
	return names
}
