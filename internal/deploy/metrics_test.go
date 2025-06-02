package deploy

import (
	"testing"
	"time"
)

func TestCalculatePRDeploymentStats(t *testing.T) {
	// Create test data
	now := time.Now()
	deployments := []DeploymentMetric{
		{
			ReleaseID:         "release-1",
			CommitSHA:         "abc123",
			PRNumber:          "123",
			CommitTime:        now.Add(-2 * time.Hour),
			ReleaseFinishTime: now.Add(-1 * time.Hour),
		},
		{
			ReleaseID:         "release-2",
			CommitSHA:         "def456",
			PRNumber:          "123", // Same PR
			CommitTime:        now.Add(-1 * time.Hour),
			ReleaseFinishTime: now,
		},
		{
			ReleaseID:         "release-3",
			CommitSHA:         "ghi789",
			PRNumber:          "456", // Different PR
			CommitTime:        now.Add(-30 * time.Minute),
			ReleaseFinishTime: now.Add(-15 * time.Minute),
		},
		{
			ReleaseID:         "release-4",
			CommitSHA:         "jkl012",
			PRNumber:          "", // Not a PR deployment
			CommitTime:        now.Add(-45 * time.Minute),
			ReleaseFinishTime: now.Add(-30 * time.Minute),
		},
	}

	stats := CalculatePRDeploymentStats(deployments)

	// Should have 2 PRs (123 and 456), ignoring the non-PR deployment
	if len(stats) != 2 {
		t.Errorf("Expected 2 PR stats, got %d", len(stats))
	}

	// Find PR 123 stats
	var pr123Stats *PRDeploymentStats
	for i := range stats {
		if stats[i].PRNumber == "123" {
			pr123Stats = &stats[i]
			break
		}
	}

	if pr123Stats == nil {
		t.Fatal("PR 123 stats not found")
	}

	// PR 123 should have 2 deployments
	if pr123Stats.DeploymentCount != 2 {
		t.Errorf("Expected PR 123 to have 2 deployments, got %d", pr123Stats.DeploymentCount)
	}

	// PR 123 should have 2 unique commit SHAs
	if len(pr123Stats.CommitSHAs) != 2 {
		t.Errorf("Expected PR 123 to have 2 unique commit SHAs, got %d", len(pr123Stats.CommitSHAs))
	}

	// Check that the first commit time and last finish time are correct
	expectedFirstCommitTime := now.Add(-2 * time.Hour)
	expectedLastFinishTime := now

	if !pr123Stats.FirstCommitTime.Equal(expectedFirstCommitTime) {
		t.Errorf("Expected first commit time %v, got %v", expectedFirstCommitTime, pr123Stats.FirstCommitTime)
	}

	if !pr123Stats.LastFinishTime.Equal(expectedLastFinishTime) {
		t.Errorf("Expected last finish time %v, got %v", expectedLastFinishTime, pr123Stats.LastFinishTime)
	}

	// Check first to last delta
	expectedDelta := 2 * time.Hour
	if pr123Stats.FirstToLastDelta != expectedDelta {
		t.Errorf("Expected first to last delta %v, got %v", expectedDelta, pr123Stats.FirstToLastDelta)
	}
}
