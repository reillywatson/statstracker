# Stats Tracker

Stats Tracker is a command-line application that provides two main tools:

1. **PR Tracker**: Analyzes GitHub pull requests and measures review times
2. **Deploy Tracker**: Measures deployment-to-log latency by tracking commit-to-log times for Google Cloud Deploy releases

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

Measures deployment-to-log latency by tracking the time between when a commit is made and when that commit shows up in Google Cloud Logger.

```bash
GITHUB_TOKEN=<mytoken> go run cmd/deploy-tracker/main.go -project <gcp-project-id>
```

**Required:**
- `GITHUB_TOKEN`: GitHub personal access token (environment variable)
- `-project`: Google Cloud project ID (command line flag)

**Optional flags:**
- `-region`: Google Cloud region (defaults to us-east4)
- `-since`: Start date in YYYY-MM-DD format (defaults to 30 days ago)
- `-until`: End date in YYYY-MM-DD format (defaults to now)

**Prerequisites for Deploy Tracker:**
- Google Cloud authentication configured (Application Default Credentials)
- Access to Google Cloud Deploy and Google Cloud Logging APIs
- Access to the EverlongProject/testenv-backend-tags and EverlongProject/services GitHub repositories
