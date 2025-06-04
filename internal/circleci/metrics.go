package circleci

import (
	"context"
	"sort"
)

// CircleCIClientInterface defines the interface for CircleCI operations
type CircleCIClientInterface interface {
	FetchFlakyTests(ctx context.Context, org, repo string) ([]FlakyTest, error)
}

// ProcessFlakyTests analyzes flaky tests and returns metrics
func ProcessFlakyTests(tests []FlakyTest) []FlakyTestMetric {
	var results []FlakyTestMetric

	for _, test := range tests {
		metric := FlakyTestMetric{
			TestName:   test.TestName,
			ClassName:  test.ClassName,
			TimesFlaky: test.TimesFlaky,
		}

		// If pipeline run information is available, extract the last occurrence time
		if test.PipelineRun != nil {
			metric.LastOccurred = &test.PipelineRun.CreatedAt
		}

		results = append(results, metric)
	}

	// Sort by times flaky (descending) for better readability
	sort.Slice(results, func(i, j int) bool {
		return results[i].TimesFlaky > results[j].TimesFlaky
	})

	return results
}
