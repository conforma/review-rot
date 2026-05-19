package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Config struct {
	GitHub  GitHubConfig  `yaml:"github"`
	Sources SourcesConfig `yaml:"sources"`
	Authors []string      `yaml:"authors"`
	Bots    []string      `yaml:"bots"`
	UI      UIConfig      `yaml:"-"`
}

type GitHubConfig struct {
	AppID          int64 `yaml:"app_id"`
	InstallationID int64 `yaml:"installation_id"`
}

type SourcesConfig struct {
	Orgs  []OrgConfig `yaml:"orgs"`
	Repos []string    `yaml:"repos"`
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
	return cfg, nil
}

func (c *Config) OrgNames() []string {
	names := make([]string, len(c.Sources.Orgs))
	for i, org := range c.Sources.Orgs {
		names[i] = org.Name
	}
	return names
}
