package github

import (
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
)

// ProcessPullRequests analyzes the pull requests and returns results
func ProcessPullRequests(client GitHubClientInterface, prs []*github.PullRequest, owner, repo string, denylist []string, tagsOwner, tagsRepo string) []PullRequestMetric {
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

		// Check if PR has associated tag commits (only if tags repo is specified)
		var tagCommits []TagCommit
		if tagsOwner != "" && tagsRepo != "" {
			tagCommits = checkPRTagCommits(client, pr, tagsOwner, tagsRepo)
		}

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
			TagCommits:        tagCommits,
		})
	}

	return results
}

// checkPRTagCommits checks if a PR has associated commits in the tags repository
// This function looks for commits in the tags repo that either:
// 1. Reference the PR number directly (pattern: pull-<number>_<sha>)
// 2. Have a branch name that matches the PR's head branch
// Returns all matching tag commits
func checkPRTagCommits(client GitHubClientInterface, pr *github.PullRequest, tagsOwner, tagsRepo string) []TagCommit {
	prNumber := pr.GetNumber()
	prBranch := ""
	if pr.GetHead() != nil {
		prBranch = pr.GetHead().GetRef()
	}

	// Fetch commits from tags repo during PR timeframe (creation to close/merge)
	startTime := pr.GetCreatedAt()
	endTime := time.Now()

	// Handle closed/merged times properly - GetMergedAt() and GetClosedAt() return time.Time, not *time.Time
	if pr.GetState() == "closed" {
		// For closed PRs, use the merge time if available, otherwise closed time
		if !pr.GetMergedAt().IsZero() {
			endTime = pr.GetMergedAt()
		} else if !pr.GetClosedAt().IsZero() {
			endTime = pr.GetClosedAt()
		} else {
			// If no close time available, extend the search window beyond creation
			endTime = pr.GetCreatedAt().Add(30 * 24 * time.Hour) // 30 days after creation
		}
	}

	commits, err := client.FetchCommits(tagsOwner, tagsRepo, startTime, endTime)
	if err != nil {
		log.Printf("Error fetching commits from tags repo for PR #%d: %v", prNumber, err)
		return []TagCommit{}
	}

	var tagCommits []TagCommit

	for _, commit := range commits {
		// Fetch the full commit with diff to analyze
		fullCommit, err := client.FetchCommit(tagsOwner, tagsRepo, commit.GetSHA())
		if err != nil {
			log.Printf("Error fetching commit %s from tags repo: %v", commit.GetSHA(), err)
			continue
		}

		// Check the commit diff for PR references
		if tagCommit := analyzeCommitDiffForPRReference(fullCommit, prNumber, prBranch); tagCommit != nil {
			tagCommits = append(tagCommits, *tagCommit)
		}
	}

	return tagCommits
}

// analyzeCommitDiffForPRReference analyzes a commit's diff to find PR references
// This function looks for two patterns in the commit diff:
// 1. Direct PR reference: pull-<pr number>_<SHA>
// 2. Branch reference: YYYY_MM_DD__HH_MM_SS__<BRANCHNAME>__<SHA>
// Returns a TagCommit if a match is found, nil otherwise
func analyzeCommitDiffForPRReference(commit *github.RepositoryCommit, prNumber int, prBranch string) *TagCommit {
	files := commit.Files
	if len(files) == 0 {
		return nil
	}

	// Patterns to match in the diff
	// Pattern 1: pull-<pr number>_<SHA> (direct PR reference)
	prPattern := regexp.MustCompile(`\+\s*\w+:\s*pull-` + strconv.Itoa(prNumber) + `_[a-f0-9]{7,40}`)

	// Pattern 2: branch name pattern (YYYY_MM_DD__HH_MM_SS__<BRANCHNAME>__<SHA>)
	var branchPattern *regexp.Regexp
	if prBranch != "" {
		branchPattern = regexp.MustCompile(`\+\s*\w+:\s*\d{4}_\d{2}_\d{2}__\d{2}_\d{2}_\d{2}__` + regexp.QuoteMeta(prBranch) + `__[a-f0-9]{7,40}`)
	}

	for _, file := range files {
		if file.Patch == nil {
			continue
		}

		patch := *file.Patch
		lines := strings.Split(patch, "\n")

		for _, line := range lines {
			if strings.HasPrefix(line, "+") {
				// Check for direct PR reference
				if matches := prPattern.FindStringSubmatch(line); matches != nil {
					return &TagCommit{
						SHA:     commit.GetSHA(),
						Message: commit.GetCommit().GetMessage(),
						Date:    commit.GetCommit().GetAuthor().GetDate(),
						Author:  commit.GetCommit().GetAuthor().GetName(),
					}
				}

				// Check for branch reference
				if branchPattern != nil {
					if matches := branchPattern.FindStringSubmatch(line); matches != nil {
						return &TagCommit{
							SHA:     commit.GetSHA(),
							Message: commit.GetCommit().GetMessage(),
							Date:    commit.GetCommit().GetAuthor().GetDate(),
							Author:  commit.GetCommit().GetAuthor().GetName(),
						}
					}
				}
			}
		}
	}

	return nil
}
