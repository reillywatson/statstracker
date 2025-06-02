package github

import (
	"log"
	"slices"
	"time"

	"github.com/google/go-github/v39/github"
)

// ProcessPullRequests analyzes the pull requests and returns results
func ProcessPullRequests(client *GitHubClient, prs []*github.PullRequest, owner, repo string, denylist []string) []PullRequestMetric {
	var results []PullRequestMetric

	// Process each PR
	for _, pr := range prs {
		// Skip draft PRs
		if pr.GetDraft() {
			continue
		}

		// Skip closed PRs that weren't merged
		if pr.GetState() == "closed" && pr.GetMergedAt().IsZero() {
			continue
		}

		prAuthorLogin := pr.GetUser().GetLogin()
		if slices.Contains(denylist, prAuthorLogin) {
			continue
		}

		reviews, err := client.FetchPullRequestReviews(owner, repo, pr.GetNumber())
		if err != nil {
			log.Printf("Error fetching reviews for PR #%d: %v", pr.GetNumber(), err)
			continue
		}

		// Track first review and first approval separately
		var firstReviewTime *time.Time
		var firstReviewer string
		var firstReviewState string

		var firstApprovalTime *time.Time
		var approver string

		var validReviewFound bool

		for _, review := range reviews {
			submittedAt := review.GetSubmittedAt()
			reviewerUser := review.GetUser().GetLogin()
			reviewState := review.GetState()

			// Skip empty, pending reviews, or self-reviews
			if reviewState == "PENDING" || reviewerUser == prAuthorLogin {
				continue
			}
			if slices.Contains(denylist, reviewerUser) {
				continue
			}

			validReviewFound = true

			// Check for first review (of any kind)
			if firstReviewTime == nil || submittedAt.Before(*firstReviewTime) {
				firstReviewTime = &submittedAt
				firstReviewer = reviewerUser
				firstReviewState = reviewState
			}

			// Check specifically for approvals
			if reviewState == "APPROVED" {
				if firstApprovalTime == nil || submittedAt.Before(*firstApprovalTime) {
					firstApprovalTime = &submittedAt
					approver = reviewerUser
				}
			}
		}

		// Calculate time to first review
		var timeToFirstReview time.Duration
		if firstReviewTime != nil {
			timeToFirstReview = firstReviewTime.Sub(pr.GetCreatedAt())
		}

		// Calculate time to first approval
		var timeToApproval time.Duration
		if firstApprovalTime != nil {
			timeToApproval = firstApprovalTime.Sub(pr.GetCreatedAt())
		}

		// Calculate time since PR was created (for PRs without reviews)
		timeSinceCreation := time.Since(pr.GetCreatedAt())

		// Always add the PR to results, but mark whether it has reviews
		results = append(results, PullRequestMetric{
			PRTitle:           pr.GetTitle(),
			PRNumber:          pr.GetNumber(),
			Author:            prAuthorLogin,
			TimeToFirstReview: timeToFirstReview,
			FirstReviewer:     firstReviewer,
			FirstReviewState:  firstReviewState,
			TimeToApproval:    timeToApproval,
			Approver:          approver,
			HasReview:         validReviewFound,
			TimeSinceCreation: timeSinceCreation,
		})
	}

	return results
}
