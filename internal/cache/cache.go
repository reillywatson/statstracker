package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Common cache errors
var (
	ErrCacheMiss = errors.New("cache miss")
)

// Cache defines the interface for all cache implementations
type Cache interface {
	// Get retrieves a value from the cache
	Get(key string, value interface{}) error

	// Set stores a value in the cache with an optional TTL
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(key string) error

	// Close cleans up the cache resources
	Close() error
}

// Entry represents a cached entry with metadata
type Entry struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// IsExpired checks if the cache entry has expired
func (e *Entry) IsExpired() bool {
	if e.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*e.ExpiresAt)
}

// CacheKeyBuilder helps build consistent cache keys
type CacheKeyBuilder struct {
	prefix string
}

func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

func (b *CacheKeyBuilder) PRKey(owner, repo string, prNumber int) string {
	return b.buildKey("pr", owner, repo, prNumber)
}

func (b *CacheKeyBuilder) PRReviewsKey(owner, repo string, prNumber int) string {
	return b.buildKey("pr_reviews", owner, repo, prNumber)
}

func (b *CacheKeyBuilder) PRsListKey(owner, repo string, startDate, endDate time.Time) string {
	start := startDate.Format("2006-01-02")
	end := endDate.Format("2006-01-02")
	return b.buildKey("prs_list", owner, repo, start, end)
}

func (b *CacheKeyBuilder) ReleaseKey(projectID, region, releaseName string) string {
	return b.buildKey("release", projectID, region, releaseName)
}

func (b *CacheKeyBuilder) RolloutsKey(projectID, region, releaseName string) string {
	return b.buildKey("rollouts", projectID, region, releaseName)
}

func (b *CacheKeyBuilder) ReleasesListKey(projectID, region, pipeline string, startDate, endDate time.Time) string {
	start := startDate.Format("2006-01-02")
	end := endDate.Format("2006-01-02")
	return b.buildKey("releases_list", projectID, region, pipeline, start, end)
}

func (b *CacheKeyBuilder) FlakyTestsKey(org, repo string) string {
	return b.buildKey("flaky-tests", org, repo)
}

func (b *CacheKeyBuilder) buildKey(parts ...interface{}) string {
	key := b.prefix
	for _, part := range parts {
		key += ":" + toString(part)
	}
	return key
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Factory function for creating default cache
func NewDefaultCache() (Cache, error) {
	return NewFileCache("statstracker")
}
