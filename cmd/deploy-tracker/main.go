package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/reillywatson/statstracker/internal/cache"
	"github.com/reillywatson/statstracker/internal/deploy"
)

func main() {
	// Define command line flags
	startDateStr := flag.String("since", "", "Start date in YYYY-MM-DD format (defaults to 30 days ago)")
	endDateStr := flag.String("until", "", "End date in YYYY-MM-DD format (defaults to now)")
	projectID := flag.String("project", "", "Google Cloud project ID (required)")
	region := flag.String("region", "us-east4", "Google Cloud region (defaults to us-east4)")
	githubOrg := flag.String("github-org", "", "GitHub organization name (required)")
	tagsRepo := flag.String("tags-repo", "", "Repository containing deployment tags (required)")
	servicesRepo := flag.String("services-repo", "", "Repository containing the actual service code (required)")

	// Parse flags
	flag.Parse()

	// Validate required parameters
	if *projectID == "" || *githubOrg == "" || *tagsRepo == "" || *servicesRepo == "" {
		fmt.Println("Usage: deploy-tracker [flags]")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("\nRequired:")
		fmt.Println("  -project: Google Cloud project ID")
		fmt.Println("  -github-org: GitHub organization name")
		fmt.Println("  -tags-repo: Repository containing deployment tags")
		fmt.Println("  -services-repo: Repository containing the actual service code")
		os.Exit(1)
	}

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
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set")
	}

	// Create cache
	cacheImpl, err := cache.NewDefaultCache()
	if err != nil {
		log.Fatalf("Error creating cache: %v", err)
	}
	defer cacheImpl.Close()

	// Create a cached Deploy client
	client, err := deploy.NewCachedDeployClient(*projectID, *region, githubToken, *githubOrg, *tagsRepo, *servicesRepo, cacheImpl)
	if err != nil {
		log.Fatalf("Error creating deploy client: %v", err)
	}
	defer client.Close()

	// Fetch test environment releases
	fmt.Printf("Fetching test environment releases for project %s in region %s from %s to %s...\n",
		*projectID, *region, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	releases, err := client.FetchTestEnvironmentReleases(startDate, endDate)
	if err != nil {
		log.Fatalf("Error fetching releases: %v", err)
	}

	fmt.Printf("Found %d test environment releases\n", len(releases))

	// Process deployments to gather results
	results := deploy.ProcessDeployments(client, releases)

	// Calculate PR deployment statistics
	prStats := deploy.CalculatePRDeploymentStats(results)

	// Print the results
	printResults(results, prStats)
}

// printResults outputs the deployment analysis results in a readable format
func printResults(results []deploy.DeploymentMetric, prStats []deploy.PRDeploymentStats) {
	if len(results) == 0 {
		fmt.Println("No deployment metrics found")
		return
	}

	// Display successful deployments
	fmt.Println("\nSuccessful Deployments (Commit to Deploy Latency):")
	fmt.Println("---------------------------------------------------")

	for _, result := range results {
		fmt.Printf("Release: %s\n", result.ReleaseID)
		fmt.Printf("  Commit SHA: %s\n", result.CommitSHA)
		if result.PRNumber != "" {
			fmt.Printf("  PR Number: %s\n", result.PRNumber)
		}
		fmt.Printf("  Commit Time: %s\n", result.CommitTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Release Start: %s\n", result.ReleaseStartTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Rollouts Completed: %s\n", result.ReleaseFinishTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Commit-to-Deploy Latency: %v\n", result.CommitToDeployLatency.Truncate(time.Second))
		fmt.Println()
	}

	printDeploymentSummaryStatistics(results)
	printPRDeploymentStatistics(prStats)
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

// printDeploymentSummaryStatistics calculates and displays mean and median deployment latencies
func printDeploymentSummaryStatistics(results []deploy.DeploymentMetric) {
	var commitToDeployLatencies []time.Duration
	var totalCommitToDeployLatency time.Duration

	for _, result := range results {
		if result.DeploymentSuccessful && result.CommitToDeployLatency > 0 {
			commitToDeployLatencies = append(commitToDeployLatencies, result.CommitToDeployLatency)
			totalCommitToDeployLatency += result.CommitToDeployLatency
		}
	}

	fmt.Println("\nDeployment Summary Statistics:")
	fmt.Println("-----------------------------")

	// Commit-to-Deploy Latency statistics
	if len(commitToDeployLatencies) > 0 {
		// Calculate mean
		meanLatency := totalCommitToDeployLatency / time.Duration(len(commitToDeployLatencies))

		// Calculate median
		medianLatency := calculateMedian(commitToDeployLatencies)

		fmt.Printf("Successful Deployments: %d\n", len(commitToDeployLatencies))
		fmt.Println("Commit-to-Deploy Latency:")
		fmt.Printf("  Mean: %v\n", meanLatency.Truncate(time.Second))
		fmt.Printf("  Median: %v\n", medianLatency.Truncate(time.Second))
	} else {
		fmt.Println("Commit-to-Deploy Latency: No data")
	}
}

// printPRDeploymentStatistics displays statistics for PR deployments
func printPRDeploymentStatistics(prStats []deploy.PRDeploymentStats) {
	if len(prStats) == 0 {
		fmt.Println("\nPR Deployment Statistics:")
		fmt.Println("------------------------")
		fmt.Println("No PR deployments found")
		return
	}

	fmt.Println("\nPR Deployment Statistics:")
	fmt.Println("------------------------")

	// Sort PRs by deployment count (descending)
	slices.SortFunc(prStats, func(a, b deploy.PRDeploymentStats) int {
		if a.DeploymentCount != b.DeploymentCount {
			return b.DeploymentCount - a.DeploymentCount // Descending order
		}
		return 0
	})

	for _, pr := range prStats {
		fmt.Printf("PR #%s:\n", pr.PRNumber)
		fmt.Printf("  Deployments: %d\n", pr.DeploymentCount)
		fmt.Printf("  Unique Commits: %d\n", len(pr.CommitSHAs))
		fmt.Printf("  First Commit: %s\n", pr.FirstCommitTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Last Deploy Finish: %s\n", pr.LastFinishTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  First Commit to Last Deploy: %v\n", pr.FirstToLastDelta.Truncate(time.Second))

		if len(pr.CommitSHAs) > 1 {
			fmt.Printf("  Commit SHAs: %v\n", pr.CommitSHAs)
		} else if len(pr.CommitSHAs) == 1 {
			fmt.Printf("  Commit SHA: %s\n", pr.CommitSHAs[0])
		}
		fmt.Println()
	}

	// Summary statistics
	var totalDeployments int
	var totalPRsWithMultipleDeployments int
	var maxDeployments int

	for _, pr := range prStats {
		totalDeployments += pr.DeploymentCount
		if pr.DeploymentCount > 1 {
			totalPRsWithMultipleDeployments++
		}
		if pr.DeploymentCount > maxDeployments {
			maxDeployments = pr.DeploymentCount
		}
	}

	fmt.Println("PR Deployment Summary:")
	fmt.Printf("  Total PRs with deployments: %d\n", len(prStats))
	fmt.Printf("  Total PR deployments: %d\n", totalDeployments)
	fmt.Printf("  PRs with multiple deployments: %d\n", totalPRsWithMultipleDeployments)
	fmt.Printf("  Maximum deployments for a single PR: %d\n", maxDeployments)
}
