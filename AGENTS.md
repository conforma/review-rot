# AI Agent Guide

## Project overview

review-rot is a PR dashboard. It has three components:

1. **Go CLI** (`cmd/review-rot/main.go`) — queries GitHub GraphQL API, outputs `data.json`
2. **Static frontend** (`frontend/`) — vanilla JS + CSS, no build step
3. **GitHub Actions** (`.github/workflows/`) — publish to GitHub Pages

## Repository structure

```text
cmd/review-rot/main.go       — CLI entry point
internal/config/              — YAML config parsing (sources + UI)
internal/github/              — GitHub API: auth, repo discovery, PR queries, filtering
internal/model/               — Output data model (Go structs with JSON tags)
frontend/index.html           — Dashboard HTML
frontend/css/style.css        — Styles
frontend/js/app.js            — Frontend logic
config/sources.yaml           — GitHub App credentials, orgs, repos, team members, bots
config/ui.yaml                — Dashboard appearance (title, logo, accent colors)
```

## Build and test

```bash
go build -o review-rot ./cmd/review-rot
go test ./...
```

## Run locally

```bash
export GITHUB_PRIVATE_KEY="$(cat private-key.pem)"
./review-rot --config-dir=config --output=data.json
```

## Publish process

`.github/workflows/publish.yaml` runs every 30 min on weekdays (and on push to main):

1. Builds the Go CLI (`go build ./cmd/review-rot`)
2. Runs it with `config/` to query GitHub and write `web/data.json`
3. Copies `frontend/*` into `web/`
4. Pushes `web/` to the `gh-pages` branch via `JamesIves/github-pages-deploy-action`

GitHub Pages is configured to serve from the `gh-pages` branch.

Authentication uses a GitHub App private key from the `GITHUB_APP_PRIVATE_KEY`
secret, passed as the `GITHUB_PRIVATE_KEY` environment variable.

## Key design decisions

- Go CLI outputs JSON consumed by a static frontend (no server)
- Pre-filtering: PRs from configured orgs always included;
  PRs from other orgs only included if author is in the `authors` list
- Bot detection: PRs from authors in the `bots` list are flagged as automated
- Frontend uses vanilla JS with no dependencies or build step
- Data refreshed every 30 minutes on weekdays via GitHub Actions cron
- Dashboard appearance (title, logo, colors) is configured via `config/ui.yaml`
  and injected into `data.json` as `ui_settings`; the frontend applies them at
  load time as CSS custom property overrides

## Customization

To adapt review-rot for a different team:

1. Edit `config/sources.yaml` with your GitHub App credentials, orgs, repos,
   and team members
2. Edit `config/ui.yaml` to set your dashboard title, logo, and accent colors
3. Set the `GITHUB_APP_PRIVATE_KEY` repository secret
4. If `ui.yaml` is omitted, defaults are used (title "Review Rot", no logo,
   neutral blue-grey palette)

## Config format

See `config/sources.yaml` for the full backend configuration. Key sections:
- `github` — App ID and installation ID
- `sources.orgs` — orgs to discover repos from
- `sources.repos` — explicit repos to monitor
- `authors` — team members for pre-filtering
- `bots` — bot accounts to flag as automated

See `config/ui.yaml` for appearance settings:
- `title` — dashboard title (shown in header and browser tab)
- `logo` — path to logo image relative to frontend directory (optional)
- `palette` — accent, accent_dark, accent_light colors
