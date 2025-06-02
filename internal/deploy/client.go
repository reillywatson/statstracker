package deploy

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	deploy "cloud.google.com/go/deploy/apiv1"
	"cloud.google.com/go/deploy/apiv1/deploypb"
	"cloud.google.com/go/logging/logadmin"
	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
)

// DeployClient wraps Google Cloud Deploy operations
type DeployClient struct {
	deployClient  *deploy.CloudDeployClient
	loggingClient *logadmin.Client
	githubClient  *github.Client
	projectID     string
	region        string
}

// NewDeployClient creates a new DeployClient with Application Default Credentials
func NewDeployClient(projectID, region, githubToken string) (*DeployClient, error) {
	ctx := context.Background()

	// Create Google Cloud Deploy client
	deployClient, err := deploy.NewCloudDeployClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create deploy client: %w", err)
	}

	// Create Google Cloud Logging client
	loggingClient, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging client: %w", err)
	}

	// Create GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: githubToken})
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)

	return &DeployClient{
		deployClient:  deployClient,
		loggingClient: loggingClient,
		githubClient:  githubClient,
		projectID:     projectID,
		region:        region,
	}, nil
}

// Close cleans up the client connections
func (c *DeployClient) Close() error {
	var errs []error

	if err := c.deployClient.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := c.loggingClient.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}

// FetchTestEnvironmentReleases gets successful releases from test environment delivery pipelines
func (c *DeployClient) FetchTestEnvironmentReleases(startDate, endDate time.Time) ([]*deploypb.Release, error) {
	ctx := context.Background()

	// First, get all delivery pipelines that contain "test"
	parent := fmt.Sprintf("projects/%s/locations/%s", c.projectID, c.region)

	req := &deploypb.ListDeliveryPipelinesRequest{
		Parent: parent,
	}

	pipelineIt := c.deployClient.ListDeliveryPipelines(ctx, req)
	var testPipelines []string

	for {
		pipeline, err := pipelineIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list delivery pipelines: %w", err)
		}

		// Check if pipeline name contains "test"
		if strings.Contains(strings.ToLower(pipeline.Name), "test4") {
			testPipelines = append(testPipelines, pipeline.Name)
		}
	}
	// print the names of the pipelines found
	if len(testPipelines) == 0 {
		return nil, fmt.Errorf("no test environment delivery pipelines found")
	}
	fmt.Printf("Found %d test environment delivery pipelines:\n", len(testPipelines))
	for _, pipeline := range testPipelines {
		fmt.Println(" -", pipeline)
	}

	var allReleases []*deploypb.Release

	// For each test pipeline, get releases
	for _, pipelineName := range testPipelines {
		fmt.Printf("Checking releases for pipeline: %s\n", pipelineName)

		releaseReq := &deploypb.ListReleasesRequest{
			Parent: pipelineName,
		}

		releaseIt := c.deployClient.ListReleases(ctx, releaseReq)
		releaseCount := 0
		filteredReleaseCount := 0

		for {
			release, err := releaseIt.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to list releases for pipeline %s: %w", pipelineName, err)
			}

			releaseCount++

			// Filter by date range
			createTime := release.CreateTime.AsTime()

			if createTime.Before(startDate) || createTime.After(endDate) {
				continue
			}

			// Only include successful releases (not failed or pending)
			if release.RenderState == deploypb.Release_SUCCEEDED {
				allReleases = append(allReleases, release)
				filteredReleaseCount++
			}
		}

		fmt.Printf("  Pipeline %s: found %d total releases, %d in date range and successful\n",
			pipelineName, releaseCount, filteredReleaseCount)
	}

	return allReleases, nil
}

// ExtractCommitSHAFromRelease extracts the commit SHA from a release
func (c *DeployClient) ExtractCommitSHAFromRelease(release *deploypb.Release) (string, time.Time, error) {
	ctx := context.Background()

	// Look for commit annotation in the release
	var commitSHA string
	if release.Annotations != nil {
		if gitSHA, exists := release.Annotations["git-sha"]; exists {
			commitSHA = gitSHA
		} else if commitURL, exists := release.Annotations["commit"]; exists {
			// Extract SHA from commit URL
			// Expected format: https://github.com/EverlongProject/testenv-backend-tags/commit/5c0119f0d4f6c0af79845df919a2389beabdeb22
			parts := strings.Split(commitURL, "/")
			if len(parts) > 0 {
				commitSHA = parts[len(parts)-1]
			}
		}
	}

	if commitSHA == "" {
		return "", time.Time{}, fmt.Errorf("no commit SHA found in release annotations")
	}

	// Get the commit from testenv-backend-tags repo
	commit, _, err := c.githubClient.Repositories.GetCommit(ctx, "EverlongProject", "testenv-backend-tags", commitSHA, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get commit from testenv-backend-tags: %w", err)
	}

	// Get the diff to extract application commit SHA
	files := commit.Files
	if len(files) == 0 {
		return "", time.Time{}, fmt.Errorf("no files in commit diff")
	}

	// Look for added lines in the diff that match our patterns
	var appCommitSHA string
	shaPattern1 := regexp.MustCompile(`\+\s*\w+:\s*pull-\d+_([a-f0-9]{7,40})`)
	shaPattern2 := regexp.MustCompile(`\+\s*\w+:\s*\d{4}_\d{2}_\d{2}__\d{2}_\d{2}_\d{2}__[^_]+__([a-f0-9]{7,40})`)

	for _, file := range files {
		if file.Patch == nil {
			continue
		}

		patch := *file.Patch
		lines := strings.Split(patch, "\n")

		for _, line := range lines {
			if strings.HasPrefix(line, "+") {
				// Try pattern 1: someapp: pull-<pr number>_SHA
				if matches := shaPattern1.FindStringSubmatch(line); len(matches) > 1 {
					appCommitSHA = matches[1]
					break
				}

				// Try pattern 2: someapp: YYYY_MM_DD__HH_MM_SS__BRANCHNAME__SHA
				if matches := shaPattern2.FindStringSubmatch(line); len(matches) > 1 {
					appCommitSHA = matches[1]
					break
				}
			}
		}

		if appCommitSHA != "" {
			break
		}
	}

	if appCommitSHA == "" {
		return "", time.Time{}, fmt.Errorf("no application commit SHA found in diff")
	}

	// Get the commit from the services repo to get the commit time
	serviceCommit, _, err := c.githubClient.Repositories.GetCommit(ctx, "EverlongProject", "services", appCommitSHA, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get commit from services repo: %w", err)
	}

	commitTime := serviceCommit.GetCommit().GetCommitter().GetDate()

	return appCommitSHA, commitTime, nil
}

// FindFirstLogEntry searches for the first log entry with the release ID
func (c *DeployClient) FindFirstLogEntry(releaseID string, searchStartTime time.Time) (time.Time, error) {
	ctx := context.Background()

	// Search for 30 minutes after the release start time
	searchEndTime := searchStartTime.Add(30 * time.Minute)

	// Build the log query
	filter := fmt.Sprintf(`labels."k8s-pod/deploy_cloud_google_com/release-id"="%s" AND timestamp>="%s" AND timestamp<="%s"`,
		releaseID,
		searchStartTime.Format(time.RFC3339),
		searchEndTime.Format(time.RFC3339))

	// Execute the query using the logadmin client
	it := c.loggingClient.Entries(ctx,
		logadmin.Filter(filter),
		logadmin.PageSize(1), // Just get the first entry
	)

	entry, err := it.Next()
	if err == iterator.Done {
		// No logs found - this means the deploy likely didn't succeed in generating logs
		return time.Time{}, fmt.Errorf("no logs found for release %s", releaseID)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("error querying logs: %w", err)
	}

	return entry.Timestamp, nil
}
