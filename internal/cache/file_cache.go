package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileCache implements Cache interface using the filesystem
type FileCache struct {
	baseDir string
}

// NewFileCache creates a new file-based cache in the OS cache directory
func NewFileCache(appName string) (*FileCache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache directory: %w", err)
	}

	baseDir := filepath.Join(cacheDir, appName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", baseDir, err)
	}

	return &FileCache{baseDir: baseDir}, nil
}

// NewFileCacheWithDir creates a new file-based cache in a specific directory
func NewFileCacheWithDir(dir string) (*FileCache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", dir, err)
	}

	return &FileCache{baseDir: dir}, nil
}

// Get retrieves a value from the cache
func (c *FileCache) Get(key string, value interface{}) error {
	filename := c.keyToFilename(key)

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check if expired
	if entry.IsExpired() {
		// Clean up expired entry
		_ = c.Delete(key)
		return ErrCacheMiss
	}

	// Unmarshal the actual data
	if err := json.Unmarshal(entry.Data, value); err != nil {
		return fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return nil
}

// Set stores a value in the cache with an optional TTL
func (c *FileCache) Set(key string, value interface{}, ttl time.Duration) error {
	// Marshal the value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Create entry
	entry := Entry{
		Data:      data,
		CreatedAt: time.Now(),
	}

	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		entry.ExpiresAt = &expiresAt
	}

	// Marshal the entry
	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Write to file
	filename := c.keyToFilename(key)

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache subdirectory: %w", err)
	}

	if err := os.WriteFile(filename, entryData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Delete removes a value from the cache
func (c *FileCache) Delete(key string) error {
	filename := c.keyToFilename(key)
	err := os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}
	return nil
}

// Close cleans up the cache resources (no-op for file cache)
func (c *FileCache) Close() error {
	return nil
}

// keyToFilename converts a cache key to a safe filename
func (c *FileCache) keyToFilename(key string) string {
	// Hash the key to ensure it's filesystem-safe and not too long
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	// Use first two characters for subdirectory to avoid too many files in one dir
	subdir := hashStr[:2]
	filename := hashStr[2:] + ".json"

	return filepath.Join(c.baseDir, subdir, filename)
}
