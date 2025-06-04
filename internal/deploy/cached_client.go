package deploy

import (
	"log"
	"time"

	"cloud.google.com/go/deploy/apiv1/deploypb"
	"github.com/reillywatson/statstracker/internal/cache"
)

// CachedDeployClient wraps DeployClient with caching capabilities
type CachedDeployClient struct {
	client *DeployClient
	cache  cache.Cache
	kb     *cache.CacheKeyBuilder
}

// NewCachedDeployClient creates a new Deploy client with caching
func NewCachedDeployClient(projectID, region, githubToken, githubOrg, tagsRepo, servicesRepo string, cacheImpl cache.Cache) (*CachedDeployClient, error) {
	client, err := NewDeployClient(projectID, region, githubToken, githubOrg, tagsRepo, servicesRepo)
	if err != nil {
		return nil, err
	}

	return &CachedDeployClient{
		client: client,
		cache:  cacheImpl,
		kb:     cache.NewCacheKeyBuilder("deploy"),
	}, nil
}

// FetchTestEnvironmentReleases fetches releases with caching
func (c *CachedDeployClient) FetchTestEnvironmentReleases(startDate, endDate time.Time) ([]*deploypb.Release, error) {
	// For release lists, we cache per-pipeline since that's how we fetch them
	// We'll need to get pipelines first, then cache each pipeline's releases
	releases, err := c.client.FetchTestEnvironmentReleases(startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Cache individual releases if they're in a cacheable state
	for _, release := range releases {
		if c.isReleaseCacheable(release) {
			releaseKey := c.kb.ReleaseKey(c.client.projectID, c.client.region, release.Name)
			if err := c.cache.Set(releaseKey, release, 24*time.Hour); err != nil {
				log.Printf("Failed to cache release %s: %v", release.Name, err)
			}
		}
	}

	return releases, nil
}

// ExtractCommitSHAFromRelease extracts commit info with caching for GitHub API calls
func (c *CachedDeployClient) ExtractCommitSHAFromRelease(release *deploypb.Release) (string, string, time.Time, error) {
	// The actual implementation delegates to the wrapped client
	// The GitHub API calls within this method will be cached if the DeployClient uses a cached GitHub client
	return c.client.ExtractCommitSHAFromRelease(release)
}

// GetReleaseFinishTime gets rollout completion time with caching
func (c *CachedDeployClient) GetReleaseFinishTime(release *deploypb.Release) (time.Time, error) {
	// Try to get rollouts from cache first
	rolloutsKey := c.kb.RolloutsKey(c.client.projectID, c.client.region, release.Name)

	var cachedResult time.Time
	if err := c.cache.Get(rolloutsKey, &cachedResult); err == nil {
		return cachedResult, nil
	} else if err != cache.ErrCacheMiss {
		log.Printf("Cache error for rollouts: %v", err)
	}

	// Cache miss, fetch from API
	finishTime, err := c.client.GetReleaseFinishTime(release)
	if err != nil {
		return time.Time{}, err
	}

	// Cache the result if the release is in a cacheable state
	if c.isReleaseCacheable(release) {
		if err := c.cache.Set(rolloutsKey, finishTime, 24*time.Hour); err != nil {
			log.Printf("Failed to cache rollouts finish time: %v", err)
		}
	}

	return finishTime, nil
}

// isReleaseCacheable determines if a release is in a state that can be cached long-term
func (c *CachedDeployClient) isReleaseCacheable(release *deploypb.Release) bool {
	if release == nil {
		return false
	}

	// Cache if release has completed successfully or failed (final states)
	return release.RenderState == deploypb.Release_SUCCEEDED ||
		release.RenderState == deploypb.Release_FAILED
}

// Close cleans up the client
func (c *CachedDeployClient) Close() error {
	defer c.cache.Close()
	return c.client.Close()
}
