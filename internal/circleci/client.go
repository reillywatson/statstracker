package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	circleAPIBaseURL = "https://circleci.com/api/v2"
	defaultTimeout   = 30 * time.Second
)

// CircleCIClient handles CircleCI API operations
type CircleCIClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// NewCircleCIClient creates a new CircleCI client
func NewCircleCIClient(token string) *CircleCIClient {
	return &CircleCIClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		token:   token,
		baseURL: circleAPIBaseURL,
	}
}

// FetchFlakyTests fetches flaky tests for a given project
func (c *CircleCIClient) FetchFlakyTests(ctx context.Context, org, repo string) ([]FlakyTest, error) {
	projectSlug := fmt.Sprintf("gh/%s/%s", org, repo)

	var allTests []FlakyTest
	nextPageToken := ""

	for {
		tests, token, err := c.fetchFlakyTestsPage(ctx, projectSlug, nextPageToken)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch flaky tests for project %s: %w", projectSlug, err)
		}

		allTests = append(allTests, tests...)

		if token == "" {
			break
		}
		nextPageToken = token
	}

	return allTests, nil
}

// fetchFlakyTestsPage fetches a single page of flaky tests
func (c *CircleCIClient) fetchFlakyTestsPage(ctx context.Context, projectSlug, pageToken string) ([]FlakyTest, string, error) {
	endpoint := fmt.Sprintf("%s/insights/%s/flaky-tests", c.baseURL, projectSlug)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")

	// Add pagination if provided
	if pageToken != "" {
		q := req.URL.Query()
		q.Add("page-token", pageToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 404 {
			return nil, "", fmt.Errorf("Project %s not found or flaky tests not available. This could mean:\n"+
				"  1. The project doesn't exist in CircleCI\n"+
				"  2. Your token doesn't have access to this project\n"+
				"  3. The project exists but doesn't have flaky test data\n"+
				"  4. The flaky tests feature is not enabled for this project\n"+
				"URL: %s", projectSlug, endpoint)
		}

		return nil, "", fmt.Errorf("API returned status %d for URL %s: %s", resp.StatusCode, endpoint, resp.Status)
	}

	var response FlakyTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.FlakyTests, response.NextPageToken, nil
}

// Close cleans up the client (no-op for HTTP client)
func (c *CircleCIClient) Close() error {
	return nil
}

// VerifyProjectAccess checks if we can access basic project information
func (c *CircleCIClient) VerifyProjectAccess(ctx context.Context, org, repo string) error {
	projectSlug := fmt.Sprintf("gh/%s/%s", org, repo)
	endpoint := fmt.Sprintf("%s/project/%s", c.baseURL, projectSlug)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("project %s not found or token doesn't have access", projectSlug)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// ListProjects lists projects that the token has access to (for debugging)
func (c *CircleCIClient) ListProjects(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/projects", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Just print the response for now since we're debugging
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Projects response: %+v\n", result)
	return nil
}
