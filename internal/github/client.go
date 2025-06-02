package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

// GitHubClientInterface defines the interface for GitHub operations
type GitHubClientInterface interface {
	FetchPullRequests(owner, repo string, startDate, endDate time.Time) ([]*github.PullRequest, error)
	FetchPullRequestReviews(owner, repo string, prNumber int) ([]*github.PullRequestReview, error)
}

type GitHubClient struct {
	client *github.Client
}

func NewGitHubClient(token string) *GitHubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &GitHubClient{
		client: github.NewClient(tc),
	}
}

func (c *GitHubClient) FetchPullRequests(owner, repo string, startDate, endDate time.Time) ([]*github.PullRequest, error) {
	ctx := context.Background()
	var allPRs []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		prs, resp, err := c.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
		}

		for _, pr := range prs {
			if !pr.GetCreatedAt().Before(startDate) && !pr.GetCreatedAt().After(endDate) {
				allPRs = append(allPRs, pr)
			}
		}

		// Break if we've processed all pages or found PRs older than our start date
		if resp.NextPage == 0 {
			break
		}

		// Check the last PR on the page - if it's older than our start date, we can stop
		lastPR := prs[len(prs)-1]
		if lastPR.GetCreatedAt().Before(startDate) {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

func (c *GitHubClient) FetchPullRequestReviews(owner, repo string, prNumber int) ([]*github.PullRequestReview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request reviews: %w", err)
	}

	return reviews, nil
}
