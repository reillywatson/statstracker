package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	ghclient "github.com/reillywatson/github-pr-tracker/internal/github"
)

func main() {
	// Define command line flags
	startDateStr := flag.String("since", "", "Start date in YYYY-MM-DD format (defaults to 30 days ago)")
	denyListStr := flag.String("denylist", "", "Comma-separated list of GitHub usernames to ignore")

	// Parse flags
	flag.Parse()

	// Check for repository argument
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: pr-tracker [flags] owner/repo")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	repoArg := args[0]
	parts := strings.Split(repoArg, "/")
	if len(parts) != 2 {
		log.Fatal("Invalid repository format. Use 'owner/repo'")
	}
	owner := parts[0]
	repo := parts[1]

	denylist := strings.Split(*denyListStr, ",")

	// Parse start date
	startDate := time.Now().AddDate(0, 0, -30) // Default to 30 days ago
	if *startDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", *startDateStr)
		if err != nil {
			log.Fatalf("Invalid date format. Please use YYYY-MM-DD: %v", err)
		}
		startDate = parsedDate
	}

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set")
	}

	// Create a GitHub client
	client := ghclient.NewGitHubClient(token)

	// Fetch pull requests with start date
	fmt.Printf("Fetching PRs for %s/%s since %s...\n", owner, repo, startDate.Format("2006-01-02"))
	prs, err := client.FetchPullRequests(owner, repo, startDate)
	if err != nil {
		log.Fatalf("Error fetching pull requests: %v", err)
	}

	// Create results to hold analysis
	type Result struct {
		PRTitle           string
		PRNumber          int
		TimeToFirstReview time.Duration
		FirstReviewer     string
		FirstReviewState  string
		TimeToApproval    time.Duration
		Approver          string
	}

	var results []Result

	fmt.Printf("Found %d pull requests for %s/%s\n", len(prs), owner, repo)

	// Process each PR
	for _, pr := range prs {
		reviews, err := client.FetchPullRequestReviews(owner, repo, pr.GetNumber())
		if err != nil {
			log.Printf("Error fetching reviews for PR #%d: %v", pr.GetNumber(), err)
			continue
		}

		// Skip PRs without reviews
		if len(reviews) == 0 {
			fmt.Printf("PR #%d: %s - No reviews yet\n", pr.GetNumber(), pr.GetTitle())
			continue
		}

		prAuthorLogin := pr.GetUser().GetLogin()
		if slices.Contains(denylist, prAuthorLogin) {
			continue
		}

		// Track first review and first approval separately
		var firstReviewTime *time.Time
		var firstReviewer string
		var firstReviewState string

		var firstApprovalTime *time.Time
		var approver string

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

		// Add result only if there's at least one review
		if firstReviewTime != nil {
			results = append(results, Result{
				PRTitle:           pr.GetTitle(),
				PRNumber:          pr.GetNumber(),
				TimeToFirstReview: timeToFirstReview,
				FirstReviewer:     firstReviewer,
				FirstReviewState:  firstReviewState,
				TimeToApproval:    timeToApproval,
				Approver:          approver,
			})
		} else {
			fmt.Printf("PR #%d: %s - No valid reviews (excluding self-reviews)\n", pr.GetNumber(), pr.GetTitle())
		}
	}

	// Output results
	if len(results) == 0 {
		fmt.Println("No pull requests with valid reviews found")
		return
	}

	fmt.Println("\nPull Request Review Times:")
	fmt.Println("---------------------------")
	for _, result := range results {
		fmt.Printf("PR #%d: %s\n", result.PRNumber, result.PRTitle)
		fmt.Printf("  Time to First Review: %v", result.TimeToFirstReview)
		fmt.Printf(" (by %s - %s)\n", result.FirstReviewer, result.FirstReviewState)

		if result.Approver != "" {
			fmt.Printf("  Time to Approval: %v", result.TimeToApproval)
			fmt.Printf(" (by %s)\n", result.Approver)
		} else {
			fmt.Printf("  Time to Approval: Not yet approved\n")
		}
		fmt.Println()
	}

	// Calculate and display averages
	var totalReviewTime time.Duration
	var reviewCount int
	var totalApprovalTime time.Duration
	var approvalCount int

	for _, result := range results {
		if result.TimeToFirstReview > 0 {
			totalReviewTime += result.TimeToFirstReview
			reviewCount++
		}

		if result.TimeToApproval > 0 {
			totalApprovalTime += result.TimeToApproval
			approvalCount++
		}
	}

	fmt.Println("\nSummary Statistics:")
	fmt.Println("-----------------")

	if reviewCount > 0 {
		avgReviewTime := totalReviewTime / time.Duration(reviewCount)
		fmt.Printf("Average Time to First Review: %v\n", avgReviewTime)
	} else {
		fmt.Println("Average Time to First Review: No data")
	}

	if approvalCount > 0 {
		avgApprovalTime := totalApprovalTime / time.Duration(approvalCount)
		fmt.Printf("Average Time to Approval: %v\n", avgApprovalTime)
	} else {
		fmt.Println("Average Time to Approval: No data")
	}
}
