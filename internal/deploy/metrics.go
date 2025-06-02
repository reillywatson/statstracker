package deploy

import (
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/deploy/apiv1/deploypb"
)

// ProcessDeployments analyzes releases and calculates commit-to-deploy latency
func ProcessDeployments(client *DeployClient, releases []*deploypb.Release) []DeploymentMetric {
	var results []DeploymentMetric

	for _, release := range releases {
		// Extract release ID from the full name
		// Format: projects/PROJECT/locations/REGION/deliveryPipelines/PIPELINE/releases/RELEASE_ID
		nameParts := strings.Split(release.Name, "/")
		var releaseID string
		if len(nameParts) > 0 {
			releaseID = nameParts[len(nameParts)-1]
		}

		// Extract commit SHA and commit time
		commitSHA, prNumber, commitTime, err := client.ExtractCommitSHAFromRelease(release)
		if err != nil {
			log.Printf("Error extracting commit SHA for release %s: %v", releaseID, err)
			continue
		}
		fmt.Println("Release ID:", releaseID, "Commit SHA:", commitSHA, "PR Number:", prNumber, "Commit Time:", commitTime)

		releaseStartTime := release.CreateTime.AsTime()

		// Get release finish time (when the last rollout completed)
		releaseFinishTime, err := client.GetReleaseFinishTime(release)
		if err != nil {
			log.Printf("Error getting release finish time for release %s: %v", releaseID, err)
			// Skip this release as it hasn't finished deploying
			continue
		}

		// Calculate commit-to-deploy latency
		commitToDeployLatency := releaseFinishTime.Sub(commitTime)

		results = append(results, DeploymentMetric{
			ReleaseID:             releaseID,
			ReleaseName:           release.Name,
			CommitSHA:             commitSHA,
			PRNumber:              prNumber,
			CommitTime:            commitTime,
			ReleaseStartTime:      releaseStartTime,
			ReleaseFinishTime:     releaseFinishTime,
			CommitToDeployLatency: commitToDeployLatency,
			DeploymentSuccessful:  true,
		})
	}

	return results
}

// CalculatePRDeploymentStats groups deployments by PR number and calculates statistics
func CalculatePRDeploymentStats(deployments []DeploymentMetric) []PRDeploymentStats {
	prMap := make(map[string][]DeploymentMetric)

	// Group deployments by PR number
	for _, deployment := range deployments {
		if deployment.PRNumber != "" { // Only include PR deployments
			prMap[deployment.PRNumber] = append(prMap[deployment.PRNumber], deployment)
		}
	}

	var stats []PRDeploymentStats

	for prNumber, prDeployments := range prMap {
		if len(prDeployments) == 0 {
			continue
		}

		// Find first commit time and last finish time
		firstCommitTime := prDeployments[0].CommitTime
		lastFinishTime := prDeployments[0].ReleaseFinishTime

		commitSHASet := make(map[string]bool)

		for _, deployment := range prDeployments {
			if deployment.CommitTime.Before(firstCommitTime) {
				firstCommitTime = deployment.CommitTime
			}
			if deployment.ReleaseFinishTime.After(lastFinishTime) {
				lastFinishTime = deployment.ReleaseFinishTime
			}
			commitSHASet[deployment.CommitSHA] = true
		}

		// Convert set to slice
		var commitSHAs []string
		for sha := range commitSHASet {
			commitSHAs = append(commitSHAs, sha)
		}

		// Calculate delta between first commit and last deploy finish
		firstToLastDelta := lastFinishTime.Sub(firstCommitTime)

		stats = append(stats, PRDeploymentStats{
			PRNumber:         prNumber,
			DeploymentCount:  len(prDeployments),
			FirstCommitTime:  firstCommitTime,
			LastFinishTime:   lastFinishTime,
			FirstToLastDelta: firstToLastDelta,
			CommitSHAs:       commitSHAs,
			Deployments:      prDeployments,
		})
	}

	return stats
}
