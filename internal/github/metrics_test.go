package github

import (
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
)

// MockGitHubClient implements GitHubClientInterface for testing
type MockGitHubClient struct {
	reviews []*github.PullRequestReview
	err     error
}

func (m *MockGitHubClient) FetchPullRequests(owner, repo string, startDate, endDate time.Time) ([]*github.PullRequest, error) {
	// Not used in ProcessPullRequests tests since PRs are passed as parameter
	return nil, nil
}

func (m *MockGitHubClient) FetchPullRequestReviews(owner, repo string, prNumber int) ([]*github.PullRequestReview, error) {
	return m.reviews, m.err
}

func TestProcessPullRequests_SkipDraftPRs(t *testing.T) {
	client := &MockGitHubClient{}

	draft := true
	user := &github.User{Login: github.String("author")}
	pr := &github.PullRequest{
		Number: github.Int(1),
		Title:  github.String("Draft PR"),
		User:   user,
		Draft:  &draft,
		State:  github.String("open"),
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for draft PR, got %d", len(results))
	}
}

func TestProcessPullRequests_SkipClosedUnmergedPRs(t *testing.T) {
	client := &MockGitHubClient{}

	user := &github.User{Login: github.String("author")}
	pr := &github.PullRequest{
		Number:   github.Int(1),
		Title:    github.String("Closed PR"),
		User:     user,
		State:    github.String("closed"),
		MergedAt: nil, // Not merged
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for closed unmerged PR, got %d", len(results))
	}
}

func TestProcessPullRequests_SkipDenylistedAuthors(t *testing.T) {
	client := &MockGitHubClient{}

	user := &github.User{Login: github.String("denylisted-author")}
	pr := &github.PullRequest{
		Number: github.Int(1),
		Title:  github.String("PR from denylisted author"),
		User:   user,
		State:  github.String("open"),
	}

	prs := []*github.PullRequest{pr}
	denylist := []string{"denylisted-author"}
	results := ProcessPullRequests(client, prs, "owner", "repo", denylist)

	if len(results) != 0 {
		t.Errorf("Expected 0 results for denylisted author, got %d", len(results))
	}
}

func TestProcessPullRequests_BasicPRWithoutReviews(t *testing.T) {
	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{}, // No reviews
	}

	user := &github.User{Login: github.String("author")}
	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("Basic PR"),
		User:      user,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.PRNumber != 1 {
		t.Errorf("Expected PR number 1, got %d", result.PRNumber)
	}
	if result.PRTitle != "Basic PR" {
		t.Errorf("Expected title 'Basic PR', got '%s'", result.PRTitle)
	}
	if result.Author != "author" {
		t.Errorf("Expected author 'author', got '%s'", result.Author)
	}
	if result.HasReview {
		t.Errorf("Expected HasReview to be false, got true")
	}
	if result.TimeToFirstReview != 0 {
		t.Errorf("Expected TimeToFirstReview to be 0, got %v", result.TimeToFirstReview)
	}
	if result.FirstReviewer != "" {
		t.Errorf("Expected FirstReviewer to be empty, got '%s'", result.FirstReviewer)
	}
}

func TestProcessPullRequests_PRWithApprovalReview(t *testing.T) {
	reviewTime := time.Now().Add(-1 * time.Hour)
	reviewer := &github.User{Login: github.String("reviewer")}

	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{
			{
				User:        reviewer,
				State:       github.String("APPROVED"),
				SubmittedAt: &reviewTime,
			},
		},
	}

	user := &github.User{Login: github.String("author")}
	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("PR with approval"),
		User:      user,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.HasReview {
		t.Errorf("Expected HasReview to be true, got false")
	}
	if result.FirstReviewer != "reviewer" {
		t.Errorf("Expected FirstReviewer to be 'reviewer', got '%s'", result.FirstReviewer)
	}
	if result.FirstReviewState != "APPROVED" {
		t.Errorf("Expected FirstReviewState to be 'APPROVED', got '%s'", result.FirstReviewState)
	}
	if result.Approver != "reviewer" {
		t.Errorf("Expected Approver to be 'reviewer', got '%s'", result.Approver)
	}

	expectedTimeToReview := reviewTime.Sub(createdAt)
	if result.TimeToFirstReview != expectedTimeToReview {
		t.Errorf("Expected TimeToFirstReview to be %v, got %v", expectedTimeToReview, result.TimeToFirstReview)
	}
	if result.TimeToApproval != expectedTimeToReview {
		t.Errorf("Expected TimeToApproval to be %v, got %v", expectedTimeToReview, result.TimeToApproval)
	}
}

func TestProcessPullRequests_PRWithMultipleReviews(t *testing.T) {
	firstReviewTime := time.Now().Add(-90 * time.Minute)
	secondReviewTime := time.Now().Add(-30 * time.Minute)

	reviewer1 := &github.User{Login: github.String("reviewer1")}
	reviewer2 := &github.User{Login: github.String("reviewer2")}

	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{
			{
				User:        reviewer2,
				State:       github.String("APPROVED"),
				SubmittedAt: &secondReviewTime,
			},
			{
				User:        reviewer1,
				State:       github.String("CHANGES_REQUESTED"),
				SubmittedAt: &firstReviewTime,
			},
		},
	}

	user := &github.User{Login: github.String("author")}
	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("PR with multiple reviews"),
		User:      user,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.HasReview {
		t.Errorf("Expected HasReview to be true, got false")
	}

	// First review should be the earliest one (changes requested)
	if result.FirstReviewer != "reviewer1" {
		t.Errorf("Expected FirstReviewer to be 'reviewer1', got '%s'", result.FirstReviewer)
	}
	if result.FirstReviewState != "CHANGES_REQUESTED" {
		t.Errorf("Expected FirstReviewState to be 'CHANGES_REQUESTED', got '%s'", result.FirstReviewState)
	}

	// First approval should be from reviewer2
	if result.Approver != "reviewer2" {
		t.Errorf("Expected Approver to be 'reviewer2', got '%s'", result.Approver)
	}

	expectedTimeToFirstReview := firstReviewTime.Sub(createdAt)
	if result.TimeToFirstReview != expectedTimeToFirstReview {
		t.Errorf("Expected TimeToFirstReview to be %v, got %v", expectedTimeToFirstReview, result.TimeToFirstReview)
	}

	expectedTimeToApproval := secondReviewTime.Sub(createdAt)
	if result.TimeToApproval != expectedTimeToApproval {
		t.Errorf("Expected TimeToApproval to be %v, got %v", expectedTimeToApproval, result.TimeToApproval)
	}
}

func TestProcessPullRequests_SkipSelfReviews(t *testing.T) {
	reviewTime := time.Now().Add(-1 * time.Hour)
	author := &github.User{Login: github.String("author")}

	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{
			{
				User:        author, // Self-review
				State:       github.String("APPROVED"),
				SubmittedAt: &reviewTime,
			},
		},
	}

	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("PR with self review"),
		User:      author,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.HasReview {
		t.Errorf("Expected HasReview to be false (self-reviews should be ignored), got true")
	}
	if result.FirstReviewer != "" {
		t.Errorf("Expected FirstReviewer to be empty, got '%s'", result.FirstReviewer)
	}
}

func TestProcessPullRequests_SkipDenylistedReviewers(t *testing.T) {
	reviewTime := time.Now().Add(-1 * time.Hour)
	reviewer := &github.User{Login: github.String("denylisted-reviewer")}

	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{
			{
				User:        reviewer,
				State:       github.String("APPROVED"),
				SubmittedAt: &reviewTime,
			},
		},
	}

	author := &github.User{Login: github.String("author")}
	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("PR with denylisted reviewer"),
		User:      author,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	denylist := []string{"denylisted-reviewer"}
	results := ProcessPullRequests(client, prs, "owner", "repo", denylist)

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.HasReview {
		t.Errorf("Expected HasReview to be false (denylisted reviewers should be ignored), got true")
	}
	if result.FirstReviewer != "" {
		t.Errorf("Expected FirstReviewer to be empty, got '%s'", result.FirstReviewer)
	}
}

func TestProcessPullRequests_SkipPendingReviews(t *testing.T) {
	reviewTime := time.Now().Add(-1 * time.Hour)
	reviewer := &github.User{Login: github.String("reviewer")}

	client := &MockGitHubClient{
		reviews: []*github.PullRequestReview{
			{
				User:        reviewer,
				State:       github.String("PENDING"),
				SubmittedAt: &reviewTime,
			},
		},
	}

	author := &github.User{Login: github.String("author")}
	createdAt := time.Now().Add(-2 * time.Hour)
	pr := &github.PullRequest{
		Number:    github.Int(1),
		Title:     github.String("PR with pending review"),
		User:      author,
		State:     github.String("open"),
		CreatedAt: &createdAt,
	}

	prs := []*github.PullRequest{pr}
	results := ProcessPullRequests(client, prs, "owner", "repo", []string{})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.HasReview {
		t.Errorf("Expected HasReview to be false (pending reviews should be ignored), got true")
	}
	if result.FirstReviewer != "" {
		t.Errorf("Expected FirstReviewer to be empty, got '%s'", result.FirstReviewer)
	}
}
