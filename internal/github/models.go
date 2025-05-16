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
