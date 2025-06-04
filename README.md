# Stats Tracker

Stats Tracker is a command-line application that provides three main tools:

1. **PR Tracker**: Analyzes GitHub pull requests and measures review times
2. **Deploy Tracker**: Measures deployment-to-log latency by tracking commit-to-log times for Google Cloud Deploy releases
3. **Flaky Tests**: Fetches and analyzes flaky tests from CircleCI for a given project

## Installation

1. Ensure you have Go installed on your machine. You can download it from [golang.org](https://golang.org/dl/).
2. Clone the repository:

   ```bash
   git clone https://github.com/reillywatson/statstracker.git
   cd statstracker
   ```

## Usage

### PR Tracker

Analyzes GitHub pull requests and measures the time taken for those PRs to be reviewed by human reviewers. It can exclude PRs opened by, or reviewed by, certain users (intended to exclude bots that review or open PRs).

```bash
GITHUB_TOKEN=<mytoken> go run cmd/pr-tracker/main.go <owner/repo>
```

Replace `<owner/repo>` with the GitHub repository you want to analyze, and GITHUB_TOKEN with a valid Github auth token.

### Deploy Tracker

Measures deployment latency by tracking the time between when a commit is made and when that commit finishes deploying.

```bash
GITHUB_TOKEN=<mytoken> go run cmd/deploy-tracker/main.go \
  -project <gcp-project-id> \
  -github-org <github-org> \
  -tags-repo <tags-repo-name> \
  -services-repo <services-repo-name>
```

**Required:**
- `GITHUB_TOKEN`: GitHub personal access token (environment variable)
- `-project`: Google Cloud project ID
- `-github-org`: GitHub organization name
- `-tags-repo`: Repository containing deployment tags
- `-services-repo`: Repository containing the actual service code

**Optional flags:**
- `-region`: Google Cloud region (defaults to us-east4)
- `-since`: Start date in YYYY-MM-DD format (defaults to 30 days ago)
- `-until`: End date in YYYY-MM-DD format (defaults to now)

**Example:**
```bash
GITHUB_TOKEN=<mytoken> go run cmd/deploy-tracker/main.go \
  -project my-gcp-project \
  -github-org someorg \
  -tags-repo some-tags-repo \
  -services-repo some-code-repo
```

**Prerequisites for Deploy Tracker:**
- Google Cloud authentication configured (Application Default Credentials)
- Access to Google Cloud Deploy API
- Access to the specified GitHub repositories

### Flaky Tests

Fetches and analyzes flaky tests from CircleCI for a specific GitHub project. Displays the tests ordered by flakiness frequency with summary statistics.

```bash
CIRCLECI_TOKEN=<mytoken> go run cmd/flaky-tests/main.go <org> <repo>
```

**Required:**
- `CIRCLECI_TOKEN`: CircleCI API token (environment variable)
- `<org>`: GitHub organization name
- `<repo>`: GitHub repository name

**Example:**
```bash
CIRCLECI_TOKEN=<mytoken> go run cmd/flaky-tests/main.go my-org my-repo
```

The command will output:
- A list of flaky tests sorted by frequency (most flaky first)
- Test names and class names where available
- Number of times each test has been flaky
- When each test was last observed as flaky (if available)
- Summary statistics including total flaky tests, total flakiness events, and averages

**Prerequisites for Flaky Tests:**
- CircleCI API token with project access
- The project must be configured in CircleCI
