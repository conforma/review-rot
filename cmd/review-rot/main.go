package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/conforma/review-rot/internal/config"
	gh "github.com/conforma/review-rot/internal/github"
	"github.com/conforma/review-rot/internal/model"
)

func main() {
	configDir := flag.String("config-dir", "", "Path to config directory containing sources.yaml and ui.yaml (required)")
	outputPath := flag.String("output", "", "Path to write data.json (required)")
	flag.Parse()

	if *configDir == "" || *outputPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: review-rot --config-dir=<path> --output=<path>\n")
		os.Exit(1)
	}

	cfg, err := config.LoadDir(*configDir)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded config: %d orgs, %d explicit repos, %d authors",
		len(cfg.Sources.Orgs), len(cfg.Sources.Repos), len(cfg.Authors))

	client, err := gh.NewClient(cfg.GitHub.AppID, cfg.GitHub.InstallationID)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	repos := gh.CollectRepos(ctx, client, cfg.OrgNames(), cfg.Sources.Repos)
	log.Printf("Monitoring %d repos", len(repos))

	var allPRs []model.PullRequest
	for _, repo := range repos {
		prs, err := gh.FetchRepoPRs(ctx, client, repo)
		if err != nil {
			log.Printf("Warning: failed to fetch PRs for %s: %v", repo, err)
			continue
		}
		allPRs = append(allPRs, prs...)
	}
	log.Printf("Fetched %d total PRs", len(allPRs))

	filtered := gh.FilterPRs(allPRs, cfg.OrgNames(), cfg.Authors)
	log.Printf("After filtering: %d PRs", len(filtered))

	if filtered == nil {
		filtered = []model.PullRequest{}
	}

	since := time.Now().UTC().AddDate(0, 0, -cfg.Leaderboard.WindowDays)
	reviewsByPerson := make(map[string][]model.ReviewedPR)
	for _, repo := range repos {
		if err := gh.FetchRepoReviewerActivity(ctx, client, repo, since, reviewsByPerson); err != nil {
			log.Printf("Warning: failed to fetch reviewer activity for %s: %v", repo, err)
			continue
		}
	}
	leaderboard := gh.BuildLeaderboard(reviewsByPerson, cfg.Leaderboard.WindowDays, since)
	log.Printf("Leaderboard: %d reviewers over the last %d days", len(leaderboard.Reviewers), cfg.Leaderboard.WindowDays)

	output := model.Output{
		GeneratedAt:  time.Now().UTC(),
		UISettings:   model.NewUISettings(cfg.UI.Title, cfg.UI.Logo, cfg.UI.Favicon, cfg.UI.Palette.Accent, cfg.UI.Palette.AccentDark, cfg.UI.Palette.AccentLight),
		PullRequests: filtered,
		Leaderboard:  leaderboard,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	if err := os.WriteFile(*outputPath, data, 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	log.Printf("Wrote %d PRs to %s", len(output.PullRequests), *outputPath)
}
