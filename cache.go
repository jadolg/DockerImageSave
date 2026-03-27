package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	cleanupInterval = 1 * time.Hour
)

// CacheManager handles the storage and cleanup of cached Docker images
type CacheManager struct {
	dir         string
	maxCacheAge time.Duration
}

// NewCacheManager creates a new CacheManager instance
func NewCacheManager(dir string, maxCacheAge time.Duration) (*CacheManager, error) {
	if dir == "" {
		tmpDir, err := os.MkdirTemp("", "docker-image-cache-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary cache directory: %w", err)
		}
		dir = tmpDir
	} else if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CacheManager{dir: dir, maxCacheAge: maxCacheAge}, nil
}

// StartCleanup starts a background goroutine that periodically removes old files
func (c *CacheManager) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	// Run initial cleanup
	c.PerformCleanup()

	for {
		select {
		case <-ticker.C:
			c.PerformCleanup()
		case <-ctx.Done():
			log.Info("Stopping cache cleanup background task")
			return
		}
	}
}

// PerformCleanup removes files from the cache directory that are older than maxCacheAge
func (c *CacheManager) PerformCleanup() {
	files, err := os.ReadDir(c.dir)
	if err != nil {
		log.WithError(err).Error("Failed to read cache directory during cleanup")
		return
	}

	now := time.Now()
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.WithField("file", file.Name()).WithError(err).Warn("Failed to get info for file during cleanup")
			continue
		}

		mtime := info.ModTime()

		if now.Sub(mtime) > c.maxCacheAge {
			path := filepath.Join(c.dir, file.Name())
			log.WithFields(log.Fields{
				"file": file.Name(),
				"age":  now.Sub(mtime),
			}).Info("Removing old cached file")
			if err := os.Remove(path); err != nil {
				log.WithField("file", file.Name()).WithError(err).Error("Failed to remove old cached file")
			}
		}
	}
}

// GetCachePath returns the full path for a cached image
func (c *CacheManager) GetCachePath(imageName string, platform Platform) string {
	return filepath.Join(c.dir, c.GetCacheFilename(imageName, platform))
}

// GetCacheFilename generates a safe filename for caching
func (c *CacheManager) GetCacheFilename(imageName string, platform Platform) string {
	return imageFilename(ParseImageReference(imageName), platform)
}

// imageFilename builds the platform-qualified tar filename for an image reference.
func imageFilename(ref ImageReference, platform Platform) string {
	parts := []string{
		sanitizeFilenameComponent(ref.Repository),
		sanitizeFilenameComponent(ref.Tag),
		sanitizeFilenameComponent(platform.OS),
		sanitizeFilenameComponent(platform.Architecture),
	}
	if platform.Variant != "" {
		parts = append(parts, sanitizeFilenameComponent(platform.Variant))
	}
	return strings.Join(parts, "_") + ".tar.gz"
}

// Dir returns the cache directory path
func (c *CacheManager) Dir() string {
	return c.dir
}
