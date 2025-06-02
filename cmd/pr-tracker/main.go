package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/reillywatson/statstracker/internal/github"
)

func main() {
	// Define command line flags
	startDateStr := flag.String("since", "", "Start date in YYYY-MM-DD format (defaults to 30 days ago)")
	endDateStr := flag.String("until", "", "End date in YYYY-MM-DD format (defaults to now)")
	denyListStr := flag.String("exclude", "", "Comma-separated list of GitHub usernames to ignore")

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
	endDate := time.Now() // Default to now
	if *endDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", *endDateStr)
		if err != nil {
			log.Fatalf("Invalid date format. Please use YYYY-MM-DD: %v", err)
		}
		endDate = parsedDate
	}
	if startDate.After(endDate) {
		log.Fatal("Start date cannot be after end date")
	}

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set")
	}

	// Create a GitHub client
	client := github.NewGitHubClient(token)

	// Fetch pull requests with start date
	fmt.Printf("Fetching PRs for %s/%s from %s to %s...\n", owner, repo, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	prs, err := client.FetchPullRequests(owner, repo, startDate, endDate)
	if err != nil {
		log.Fatalf("Error fetching pull requests: %v", err)
	}

	fmt.Printf("Found %d pull requests for %s/%s\n", len(prs), owner, repo)

	// Process pull requests to gather results
	results := github.ProcessPullRequests(client, prs, owner, repo, denylist)

	// Print the results
	printResults(results)
}

// printResults outputs the analysis results in a readable format
func printResults(results []github.PullRequestMetric) {
	// Output results
	if len(results) == 0 {
		fmt.Println("No pull requests found")
		return
	}

	// First, display PRs with reviews
	fmt.Println("\nPull Requests With Reviews:")
	fmt.Println("---------------------------")

	reviewedPRsCount := 0
	for _, result := range results {
		if result.HasReview {
			reviewedPRsCount++
			fmt.Printf("PR #%d: %s\n", result.PRNumber, result.PRTitle)
			fmt.Printf("  Time to First Review: %v", result.TimeToFirstReview.Truncate(time.Second))
			fmt.Printf(" (by %s - %s)\n", result.FirstReviewer, result.FirstReviewState)

			if result.Approver != "" {
				fmt.Printf("  Time to Approval: %v", result.TimeToApproval.Truncate(time.Second))
				fmt.Printf(" (by %s)\n", result.Approver)
			} else {
				fmt.Printf("  Time to Approval: Not yet approved\n")
			}
			fmt.Println()
		}
	}

	if reviewedPRsCount == 0 {
		fmt.Println("  None found")
	}

	// Then, display PRs without reviews
	fmt.Println("\nPull Requests Awaiting Review:")
	fmt.Println("------------------------------")

	awaitingReviewCount := 0
	for _, result := range results {
		if !result.HasReview {
			awaitingReviewCount++
			fmt.Printf("PR #%d: %s\n", result.PRNumber, result.PRTitle)
			fmt.Printf("Author: %s\n", result.Author)
			fmt.Printf("  Waiting for: %v\n", result.TimeSinceCreation.Truncate(time.Second))
			fmt.Println()
		}
	}

	if awaitingReviewCount == 0 {
		fmt.Println("  None found")
	}

	printSummaryStatistics(results)
}

// calculateMedian calculates the median of a slice of time.Duration
func calculateMedian(durations []time.Duration) time.Duration {
	n := len(durations)
	if n == 0 {
		return 0
	}

	// Sort the slice
	slices.Sort(durations)

	// If odd, return the middle element
	if n%2 != 0 {
		return durations[n/2]
	}

	// If even, return the average of the two middle elements
	mid1 := durations[(n/2)-1]
	mid2 := durations[n/2]
	return (mid1 + mid2) / 2
}

// printSummaryStatistics calculates and displays mean and median review times
func printSummaryStatistics(results []github.PullRequestMetric) {
	// Collect all the time durations for each category
	var firstReviewTimes []time.Duration
	var approvalTimes []time.Duration
	var waitingTimes []time.Duration

	// Calculate totals for means
	var totalReviewTime time.Duration
	var totalApprovalTime time.Duration
	var totalWaitingTime time.Duration

	for _, result := range results {
		if result.HasReview {
			if result.TimeToFirstReview > 0 {
				firstReviewTimes = append(firstReviewTimes, result.TimeToFirstReview)
				totalReviewTime += result.TimeToFirstReview
			}

			if result.TimeToApproval > 0 {
				approvalTimes = append(approvalTimes, result.TimeToApproval)
				totalApprovalTime += result.TimeToApproval
			}
		} else {
			// Track PRs with no reviews
			waitingTimes = append(waitingTimes, result.TimeSinceCreation)
			totalWaitingTime += result.TimeSinceCreation
		}
	}

	fmt.Println("\nSummary Statistics:")
	fmt.Println("-----------------")

	// Time to First Review statistics
	if len(firstReviewTimes) > 0 {
		// Calculate mean
		meanReviewTime := totalReviewTime / time.Duration(len(firstReviewTimes))

		// Calculate median
		medianReviewTime := calculateMedian(firstReviewTimes)

		fmt.Println("Time to First Review:")
		fmt.Printf("  Mean: %v\n", meanReviewTime.Truncate(time.Second))
		fmt.Printf("  Median: %v\n", medianReviewTime.Truncate(time.Second))
	} else {
		fmt.Println("Time to First Review: No data")
	}

	// Time to Approval statistics
	if len(approvalTimes) > 0 {
		// Calculate mean
		meanApprovalTime := totalApprovalTime / time.Duration(len(approvalTimes))

		// Calculate median
		medianApprovalTime := calculateMedian(approvalTimes)

		fmt.Println("Time to Approval:")
		fmt.Printf("  Mean: %v\n", meanApprovalTime.Truncate(time.Second))
		fmt.Printf("  Median: %v\n", medianApprovalTime.Truncate(time.Second))
	} else {
		fmt.Println("Time to Approval: No data")
	}

	// PRs awaiting review statistics
	if len(waitingTimes) > 0 {
		// Calculate mean
		meanWaitingTime := totalWaitingTime / time.Duration(len(waitingTimes))

		// Calculate median
		medianWaitingTime := calculateMedian(waitingTimes)

		fmt.Printf("PRs Awaiting Review: %d\n", len(waitingTimes))
		fmt.Printf("  Mean wait time: %v\n", meanWaitingTime.Truncate(time.Second))
		fmt.Printf("  Median wait time: %v\n", medianWaitingTime.Truncate(time.Second))
	} else {
		fmt.Println("PRs Awaiting Review: 0")
	}
}
