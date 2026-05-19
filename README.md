# review-rot

PR dashboard that shows open pull requests across monitored GitHub repositories
so team members can see at a glance which PRs need attention.

Live example: **[conforma.dev/review-rot](https://conforma.dev/review-rot/)**

## How it works

A Go CLI queries the GitHub GraphQL API for open PRs across configured repos,
enriches each with review/CI/conversation metadata, and outputs `data.json`. A
static frontend (vanilla JS + CSS) renders the data with client-side filtering
and sorting.

## Deployment

A GitHub Actions workflow (`.github/workflows/publish.yaml`) runs every
30 minutes on weekdays:

1. Builds the Go CLI
2. Runs it with the `config/` directory to produce `web/data.json`
3. Copies the static frontend files into `web/`
4. Pushes `web/` to the `gh-pages` branch

GitHub Pages is configured to serve from the `gh-pages` branch.

The CLI authenticates as a GitHub App using a private key passed via the
`GITHUB_PRIVATE_KEY` environment variable. In this repository the workflow
reads it from the `EC_AUTOMATION_KEY` secret.

## Configuration

The `config/` directory contains two files:

- **`sources.yaml`** — GitHub App credentials, monitored orgs/repos, team
  members, and bot accounts. See the comments in that file for details.
- **`ui.yaml`** — Dashboard appearance: title, logo, and accent colors.

## GitHub App

The CLI uses a GitHub App for authentication. GitHub's GraphQL API requires
authentication even for public repositories, and a GitHub App provides its own
rate limit (5,000+ requests/hour) without being tied to any individual's
account.

The app only needs **read-only** access to pull requests and repository
metadata — no write permissions are required. See the
[GitHub docs](https://docs.github.com/en/apps/creating-github-apps) for
instructions on creating and installing a GitHub App.

## Customization

To use review-rot for your own team:

1. Fork the repository
2. Create a GitHub App (see above) and install it on your org
3. Edit `config/sources.yaml` with your App ID, Installation ID, orgs, repos,
   and team members
4. Edit `config/ui.yaml` to set your team name, logo, and brand colors
5. Add the App's private key as a repository secret and update the
   `GITHUB_PRIVATE_KEY` env var in `.github/workflows/publish.yaml` to
   reference it (this repo uses `EC_AUTOMATION_KEY`)
6. Configure GitHub Pages to serve from the `gh-pages` branch

If `ui.yaml` is omitted, the dashboard uses the default title ("Review Rot"),
no logo, and a neutral blue-grey color scheme.
