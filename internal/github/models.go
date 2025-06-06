package github

import "time"

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsDraft   bool      `json:"is_draft"`
	Author    string    `json:"author"`
	Reviews   []Review  `json:"reviews"`
}

// Review represents a review on a pull request.
type Review struct {
	ID     int       `json:"id"`
	User   string    `json:"user"`
	Status string    `json:"status"`
	Date   time.Time `json:"date"`
}

// TagCommit represents a commit in the tags repository that references a PR
type TagCommit struct {
	SHA     string    // The commit SHA in the tags repo
	Message string    // The commit message
	Date    time.Time // When the commit was created
	Author  string    // The commit author
}

// PullRequestMetric represents the analysis results for a single PR
type PullRequestMetric struct {
	PRTitle           string
	PRNumber          int
	Author            string
	TimeToFirstReview time.Duration
	FirstReviewer     string
	FirstReviewState  string
	TimeToApproval    time.Duration
	Approver          string
	HasReview         bool          // Flag to indicate if PR has at least one review
	TimeSinceCreation time.Duration // How long the PR has been open without review
	TagCommits        []TagCommit   // All tag commits that reference this PR
}
