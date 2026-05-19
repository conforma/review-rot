package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
github:
  app_id: 245286
  installation_id: 59973090
sources:
  orgs:
    - name: conforma
    - name: enterprise-contract
  repos:
    - konflux-ci/build-definitions
    - tektoncd/chains
authors:
  - simonbaird
  - st3penta
`
	path := writeFile(t, "config.yaml", content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.GitHub.AppID != 245286 {
		t.Errorf("AppID = %d, want 245286", cfg.GitHub.AppID)
	}
	if cfg.GitHub.InstallationID != 59973090 {
		t.Errorf("InstallationID = %d, want 59973090", cfg.GitHub.InstallationID)
	}
	if len(cfg.Sources.Orgs) != 2 {
		t.Errorf("Orgs count = %d, want 2", len(cfg.Sources.Orgs))
	}
	if cfg.Sources.Orgs[0].Name != "conforma" {
		t.Errorf("Orgs[0].Name = %q, want %q", cfg.Sources.Orgs[0].Name, "conforma")
	}
	if len(cfg.Sources.Repos) != 2 {
		t.Errorf("Repos count = %d, want 2", len(cfg.Sources.Repos))
	}
	if len(cfg.Authors) != 2 {
		t.Errorf("Authors count = %d, want 2", len(cfg.Authors))
	}
	orgNames := cfg.OrgNames()
	if len(orgNames) != 2 || orgNames[0] != "conforma" || orgNames[1] != "enterprise-contract" {
		t.Errorf("OrgNames() = %v, want [conforma enterprise-contract]", orgNames)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()

	sources := `
github:
  app_id: 123
  installation_id: 456
sources:
  orgs:
    - name: myorg
authors:
  - alice
`
	ui := `
title: My Dashboard
logo: images/logo.png
favicon: images/favicon.ico
palette:
  accent: "#ff0000"
  accent_dark: "#cc0000"
  accent_light: "#ffeeee"
`
	if err := os.WriteFile(filepath.Join(dir, "sources.yaml"), []byte(sources), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ui.yaml"), []byte(ui), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error: %v", err)
	}

	if cfg.GitHub.AppID != 123 {
		t.Errorf("AppID = %d, want 123", cfg.GitHub.AppID)
	}
	if cfg.Sources.Orgs[0].Name != "myorg" {
		t.Errorf("Org = %q, want myorg", cfg.Sources.Orgs[0].Name)
	}
	if cfg.UI.Title != "My Dashboard" {
		t.Errorf("UI.Title = %q, want %q", cfg.UI.Title, "My Dashboard")
	}
	if cfg.UI.Logo != "images/logo.png" {
		t.Errorf("UI.Logo = %q, want %q", cfg.UI.Logo, "images/logo.png")
	}
	if cfg.UI.Favicon != "images/favicon.ico" {
		t.Errorf("UI.Favicon = %q, want %q", cfg.UI.Favicon, "images/favicon.ico")
	}
	if cfg.UI.Palette.Accent != "#ff0000" {
		t.Errorf("UI.Palette.Accent = %q, want %q", cfg.UI.Palette.Accent, "#ff0000")
	}
	if cfg.UI.Palette.AccentDark != "#cc0000" {
		t.Errorf("UI.Palette.AccentDark = %q, want %q", cfg.UI.Palette.AccentDark, "#cc0000")
	}
	if cfg.UI.Palette.AccentLight != "#ffeeee" {
		t.Errorf("UI.Palette.AccentLight = %q, want %q", cfg.UI.Palette.AccentLight, "#ffeeee")
	}
}

func TestLoadDirWithoutUI(t *testing.T) {
	dir := t.TempDir()

	sources := `
github:
  app_id: 123
  installation_id: 456
sources:
  orgs:
    - name: myorg
`
	if err := os.WriteFile(filepath.Join(dir, "sources.yaml"), []byte(sources), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error: %v", err)
	}

	if cfg.GitHub.AppID != 123 {
		t.Errorf("AppID = %d, want 123", cfg.GitHub.AppID)
	}
	if cfg.UI.Title != "" {
		t.Errorf("UI.Title = %q, want empty", cfg.UI.Title)
	}
}

func TestLoadMissingAppID(t *testing.T) {
	content := `
github:
  installation_id: 59973090
sources:
  orgs:
    - name: conforma
`
	path := writeFile(t, "config.yaml", content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should fail with missing app_id")
	}
	if !strings.Contains(err.Error(), "app_id") {
		t.Errorf("error should mention app_id, got: %v", err)
	}
}

func TestLoadMissingInstallationID(t *testing.T) {
	content := `
github:
  app_id: 245286
sources:
  orgs:
    - name: conforma
`
	path := writeFile(t, "config.yaml", content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should fail with missing installation_id")
	}
	if !strings.Contains(err.Error(), "installation_id") {
		t.Errorf("error should mention installation_id, got: %v", err)
	}
}

func TestLoadMissingSources(t *testing.T) {
	content := `
github:
  app_id: 245286
  installation_id: 59973090
`
	path := writeFile(t, "config.yaml", content)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should fail with no orgs or repos")
	}
	if !strings.Contains(err.Error(), "org") || !strings.Contains(err.Error(), "repo") {
		t.Errorf("error should mention org/repo, got: %v", err)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("Load() should fail with nonexistent file")
	}
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
