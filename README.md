# GitHub PR Tracker

GitHub PR Tracker is a command-line application that allows users to pull pull requests (PRs) from a specified GitHub repository and measure the time taken for those PRs to be reviewed by human reviewers. It can exclude PRs opened by, or reviewed by, certain users (intended to exclude bots that review or open PRs).

## Installation

1. Ensure you have Go installed on your machine. You can download it from [golang.org](https://golang.org/dl/).
2. Clone the repository:

   ```bash
   git clone https://github.com/reillywatson/github-pr-tracker.git
   cd github-pr-tracker
   ```

## Usage

To run the application, use the following command:

```bash
GITHUB_TOKEN=<mytoken> go run cmd/pr-tracker/main.go <owner/repo>
```

Replace `<owner/repo>` with the GitHub repository you want to analyze, and GITHUB_TOKEN with a valid Github auth token.
