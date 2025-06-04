package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/reillywatson/statstracker/internal/cache"
	"github.com/reillywatson/statstracker/internal/circleci"
)

func main() {
	// Define command line flags
	flag.Parse()

	// Check for org and repo arguments
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: flaky-tests <org> <repo>")
		fmt.Println("Example: flaky-tests my-org my-repo")
		fmt.Println("\nRequired environment variables:")
		fmt.Println("  CIRCLECI_TOKEN: CircleCI API token")
		os.Exit(1)
	}

	org := args[0]
	repo := args[1]

	// Get CircleCI token from environment
	token := os.Getenv("CIRCLECI_TOKEN")
	if token == "" {
		log.Fatal("CIRCLECI_TOKEN environment variable not set")
	}

	// Create cache
	cacheImpl, err := cache.NewDefaultCache()
	if err != nil {
		log.Fatalf("Error creating cache: %v", err)
	}
	defer cacheImpl.Close()

	// Create a cached CircleCI client
	client := circleci.NewCachedCircleCIClient(token, cacheImpl)
	defer client.Close()

	ctx := context.Background()

	// First verify we can access the project
	fmt.Printf("Verifying access to project %s/%s...\n", org, repo)
	if err := client.VerifyProjectAccess(ctx, org, repo); err != nil {
		log.Fatalf("Error accessing project: %v", err)
	}
	fmt.Println("✓ Project access verified")

	// Fetch flaky tests
	fmt.Printf("Fetching flaky tests for %s/%s...\n", org, repo)
	tests, err := client.FetchFlakyTests(ctx, org, repo)
	if err != nil {
		log.Fatalf("Error fetching flaky tests: %v", err)
	}

	fmt.Printf("Found %d flaky tests for %s/%s\n", len(tests), org, repo)

	// Process flaky tests to gather metrics
	results := circleci.ProcessFlakyTests(tests)

	// Print the results
	printResults(results)
}

// printResults outputs the flaky test analysis results in a readable format
func printResults(results []circleci.FlakyTestMetric) {
	if len(results) == 0 {
		fmt.Println("No flaky tests found")
		return
	}

	fmt.Println("\nFlaky Tests (sorted by frequency):")
	fmt.Println("==================================")

	for _, result := range results {
		fmt.Printf("Test: %s\n", result.TestName)
		if result.ClassName != "" {
			fmt.Printf("  Class: %s\n", result.ClassName)
		}
		fmt.Printf("  Times Flaky: %d\n", result.TimesFlaky)
		if result.LastOccurred != nil {
			fmt.Printf("  Last Occurred: %s\n", result.LastOccurred.Format("2006-01-02 15:04:05 MST"))
		}
		fmt.Println()
	}

	printSummaryStatistics(results)
}

// printSummaryStatistics calculates and displays summary statistics
func printSummaryStatistics(results []circleci.FlakyTestMetric) {
	if len(results) == 0 {
		return
	}

	// Calculate total flakiness count
	totalFlakiness := 0
	var flakinessValues []int

	for _, result := range results {
		totalFlakiness += result.TimesFlaky
		flakinessValues = append(flakinessValues, result.TimesFlaky)
	}

	// Calculate mean
	meanFlakiness := float64(totalFlakiness) / float64(len(results))

	// Calculate median
	slices.Sort(flakinessValues)
	var medianFlakiness float64
	n := len(flakinessValues)
	if n%2 == 0 {
		medianFlakiness = float64(flakinessValues[n/2-1]+flakinessValues[n/2]) / 2.0
	} else {
		medianFlakiness = float64(flakinessValues[n/2])
	}

	fmt.Println("Summary Statistics:")
	fmt.Println("------------------")
	fmt.Printf("Total Flaky Tests: %d\n", len(results))
	fmt.Printf("Total Flakiness Events: %d\n", totalFlakiness)
	fmt.Printf("Average Flakiness per Test: %.1f\n", meanFlakiness)
	fmt.Printf("Median Flakiness per Test: %.1f\n", medianFlakiness)
	fmt.Printf("Most Flaky Test: %d occurrences\n", flakinessValues[len(flakinessValues)-1])
	fmt.Printf("Least Flaky Test: %d occurrences\n", flakinessValues[0])
}
