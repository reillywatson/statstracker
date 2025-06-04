package circleci

import (
	"context"
	"time"

	"github.com/reillywatson/statstracker/internal/cache"
)

// CachedCircleCIClient wraps CircleCIClient with caching capabilities
type CachedCircleCIClient struct {
	client *CircleCIClient
	cache  cache.Cache
	kb     *cache.CacheKeyBuilder
}

// NewCachedCircleCIClient creates a new CircleCI client with caching
func NewCachedCircleCIClient(token string, cacheImpl cache.Cache) *CachedCircleCIClient {
	return &CachedCircleCIClient{
		client: NewCircleCIClient(token),
		cache:  cacheImpl,
		kb:     cache.NewCacheKeyBuilder("circleci"),
	}
}

// FetchFlakyTests fetches flaky tests with caching
func (c *CachedCircleCIClient) FetchFlakyTests(ctx context.Context, org, repo string) ([]FlakyTest, error) {
	// Create cache key using the key builder
	key := c.kb.FlakyTestsKey(org, repo)

	// Try to get from cache
	var cachedTests []FlakyTest
	if err := c.cache.Get(key, &cachedTests); err == nil {
		return cachedTests, nil
	} else if err != cache.ErrCacheMiss {
		// Log non-miss errors but continue
	}

	// Not in cache or cache miss, fetch from API
	tests, err := c.client.FetchFlakyTests(ctx, org, repo)
	if err != nil {
		return nil, err
	}

	// Store in cache with 1 hour TTL (flaky tests can change frequently)
	if err := c.cache.Set(key, tests, 1*time.Hour); err != nil {
		// Log error but don't fail the request
	}

	return tests, nil
}

// VerifyProjectAccess checks if we can access basic project information (no caching)
func (c *CachedCircleCIClient) VerifyProjectAccess(ctx context.Context, org, repo string) error {
	return c.client.VerifyProjectAccess(ctx, org, repo)
}

// Close cleans up the client connections
func (c *CachedCircleCIClient) Close() error {
	return c.client.Close()
}
