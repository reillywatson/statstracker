package github

import (
	"log"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/reillywatson/statstracker/internal/cache"
)

// CachedGitHubClient wraps GitHubClient with caching capabilities
type CachedGitHubClient struct {
	client *GitHubClient
	cache  cache.Cache
	kb     *cache.CacheKeyBuilder
}

// NewCachedGitHubClient creates a new GitHub client with caching
func NewCachedGitHubClient(token string, cacheImpl cache.Cache) *CachedGitHubClient {
	return &CachedGitHubClient{
		client: NewGitHubClient(token),
		cache:  cacheImpl,
		kb:     cache.NewCacheKeyBuilder("github"),
	}
}

// FetchPullRequests fetches pull requests with caching
func (c *CachedGitHubClient) FetchPullRequests(owner, repo string, startDate, endDate time.Time) ([]*github.PullRequest, error) {
	// Try to get from cache first
	cacheKey := c.kb.PRsListKey(owner, repo, startDate, endDate)
	var cachedPRs []*github.PullRequest
	if err := c.cache.Get(cacheKey, &cachedPRs); err == nil {
		return cachedPRs, nil
	} else if err != cache.ErrCacheMiss {
		log.Printf("Cache error for PRs list: %v", err)
	}

	// Cache miss, fetch from API
	prs, err := c.client.FetchPullRequests(owner, repo, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Cache the result - use longer TTL for historical data, shorter for recent data
	ttl := c.calculatePRListTTL(endDate)
	if err := c.cache.Set(cacheKey, prs, ttl); err != nil {
		log.Printf("Failed to cache PRs list: %v", err)
	}

	// Also cache individual PRs if they're in a cacheable state
	for _, pr := range prs {
		if c.isPRCacheable(pr) {
			prKey := c.kb.PRKey(owner, repo, pr.GetNumber())
			if err := c.cache.Set(prKey, pr, 24*time.Hour); err != nil {
				log.Printf("Failed to cache individual PR #%d: %v", pr.GetNumber(), err)
			}
		}
	}

	return prs, nil
}

// FetchPullRequestReviews fetches PR reviews with caching
func (c *CachedGitHubClient) FetchPullRequestReviews(owner, repo string, prNumber int) ([]*github.PullRequestReview, error) {
	// Try to get from cache first
	cacheKey := c.kb.PRReviewsKey(owner, repo, prNumber)
	var cachedReviews []*github.PullRequestReview
	if err := c.cache.Get(cacheKey, &cachedReviews); err == nil {
		return cachedReviews, nil
	} else if err != cache.ErrCacheMiss {
		log.Printf("Cache error for PR #%d reviews: %v", prNumber, err)
	}

	// Cache miss, fetch from API
	reviews, err := c.client.FetchPullRequestReviews(owner, repo, prNumber)
	if err != nil {
		return nil, err
	}

	// Check if PR is in a cacheable state
	var pr *github.PullRequest
	prKey := c.kb.PRKey(owner, repo, prNumber)
	if err := c.cache.Get(prKey, &pr); err == nil && c.isPRCacheable(pr) {
		// PR is cacheable, cache reviews with longer TTL
		if err := c.cache.Set(cacheKey, reviews, 24*time.Hour); err != nil {
			log.Printf("Failed to cache PR #%d reviews: %v", prNumber, err)
		}
	} else {
		// PR might still be active, cache with shorter TTL
		if err := c.cache.Set(cacheKey, reviews, 1*time.Hour); err != nil {
			log.Printf("Failed to cache PR #%d reviews: %v", prNumber, err)
		}
	}

	return reviews, nil
}

// isPRCacheable determines if a PR is in a state that can be cached long-term
func (c *CachedGitHubClient) isPRCacheable(pr *github.PullRequest) bool {
	if pr == nil {
		return false
	}

	// Cache if PR is closed (merged or not)
	return pr.GetState() == "closed"
}

// calculatePRListTTL calculates TTL for PR list cache based on how recent the data is
func (c *CachedGitHubClient) calculatePRListTTL(endDate time.Time) time.Duration {
	daysSinceEnd := time.Since(endDate).Hours() / 24

	// Historical data (older than 7 days): cache for 24 hours
	if daysSinceEnd > 7 {
		return 24 * time.Hour
	}

	// Recent data (last 7 days): cache for 1 hour
	return 1 * time.Hour
}

// Close cleans up the client
func (c *CachedGitHubClient) Close() error {
	return c.cache.Close()
}
