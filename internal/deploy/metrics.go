package deploy

import (
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/deploy/apiv1/deploypb"
)

// ProcessDeployments analyzes releases and calculates commit-to-log latency
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
		commitSHA, commitTime, err := client.ExtractCommitSHAFromRelease(release)
		if err != nil {
			log.Printf("Error extracting commit SHA for release %s: %v", releaseID, err)
			continue
		}
		fmt.Println("Release ID:", releaseID, "Commit SHA:", commitSHA, "Commit Time:", commitTime)

		releaseStartTime := release.CreateTime.AsTime()

		// Search for first log entry
		firstLogTime, err := client.FindFirstLogEntry(releaseID, releaseStartTime)
		if err != nil {
			log.Printf("No logs found for release %s: %v", releaseID, err)
			// Skip this release as requested
			continue
		}

		// Calculate commit-to-log latency
		commitToLogLatency := firstLogTime.Sub(commitTime)

		results = append(results, DeploymentMetric{
			ReleaseID:            releaseID,
			ReleaseName:          release.Name,
			CommitSHA:            commitSHA,
			CommitTime:           commitTime,
			ReleaseStartTime:     releaseStartTime,
			FirstLogTime:         firstLogTime,
			CommitToLogLatency:   commitToLogLatency,
			DeploymentSuccessful: true,
		})
	}

	return results
}
