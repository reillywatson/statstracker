package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/reillywatson/statstracker/internal/deploy"
)

func main() {
	// Define command line flags
	startDateStr := flag.String("since", "", "Start date in YYYY-MM-DD format (defaults to 30 days ago)")
	endDateStr := flag.String("until", "", "End date in YYYY-MM-DD format (defaults to now)")
	projectID := flag.String("project", "", "Google Cloud project ID (required)")
	region := flag.String("region", "us-east4", "Google Cloud region (defaults to us-east4)")

	// Parse flags
	flag.Parse()

	// Validate required parameters
	if *projectID == "" {
		fmt.Println("Usage: deploy-tracker [flags]")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("\nRequired:")
		fmt.Println("  -project: Google Cloud project ID")
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

	// Create a Deploy client
	client, err := deploy.NewDeployClient(*projectID, *region, githubToken)
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

	// Print the results
	printResults(results)
}

// printResults outputs the deployment analysis results in a readable format
func printResults(results []deploy.DeploymentMetric) {
	if len(results) == 0 {
		fmt.Println("No deployment metrics found")
		return
	}

	// Display successful deployments
	fmt.Println("\nSuccessful Deployments (Commit to Log Latency):")
	fmt.Println("------------------------------------------------")

	for _, result := range results {
		fmt.Printf("Release: %s\n", result.ReleaseID)
		fmt.Printf("  Commit SHA: %s\n", result.CommitSHA)
		fmt.Printf("  Commit Time: %s\n", result.CommitTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Release Start: %s\n", result.ReleaseStartTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  First Log: %s\n", result.FirstLogTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Commit-to-Log Latency: %v\n", result.CommitToLogLatency.Truncate(time.Second))
		fmt.Println()
	}

	printDeploymentSummaryStatistics(results)
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
	var commitToLogLatencies []time.Duration
	var totalCommitToLogLatency time.Duration

	for _, result := range results {
		if result.DeploymentSuccessful && result.CommitToLogLatency > 0 {
			commitToLogLatencies = append(commitToLogLatencies, result.CommitToLogLatency)
			totalCommitToLogLatency += result.CommitToLogLatency
		}
	}

	fmt.Println("\nDeployment Summary Statistics:")
	fmt.Println("-----------------------------")

	// Commit-to-Log Latency statistics
	if len(commitToLogLatencies) > 0 {
		// Calculate mean
		meanLatency := totalCommitToLogLatency / time.Duration(len(commitToLogLatencies))

		// Calculate median
		medianLatency := calculateMedian(commitToLogLatencies)

		fmt.Printf("Successful Deployments: %d\n", len(commitToLogLatencies))
		fmt.Println("Commit-to-Log Latency:")
		fmt.Printf("  Mean: %v\n", meanLatency.Truncate(time.Second))
		fmt.Printf("  Median: %v\n", medianLatency.Truncate(time.Second))
	} else {
		fmt.Println("Commit-to-Log Latency: No data")
	}
}
