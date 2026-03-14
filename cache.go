package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
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
			log.Println("Stopping cache cleanup background task")
			return
		}
	}
}

// PerformCleanup removes files from the cache directory that are older than maxCacheAge
func (c *CacheManager) PerformCleanup() {
	files, err := os.ReadDir(c.dir)
	if err != nil {
		log.Printf("Failed to read cache directory during cleanup: %v\n", err)
		return
	}

	now := time.Now()
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.Printf("Failed to get info for file %s during cleanup: %v\n", file.Name(), err)
			continue
		}

		stat := info.Sys().(*syscall.Stat_t)
		atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))

		if now.Sub(atime) > c.maxCacheAge {
			path := filepath.Join(c.dir, file.Name())
			log.Printf("Removing old cached file: %s (age: %v)\n", file.Name(), now.Sub(atime))
			if err := os.Remove(path); err != nil {
				log.Printf("Failed to remove old cached file %s: %v\n", file.Name(), err)
			}
		}
	}
}

// GetCachePath returns the full path for a cached image
func (c *CacheManager) GetCachePath(imageName string) string {
	return filepath.Join(c.dir, c.GetCacheFilename(imageName))
}

// GetCacheFilename generates a safe filename for caching
func (c *CacheManager) GetCacheFilename(imageName string) string {
	ref := ParseImageReference(imageName)
	safeImageName := sanitizeFilenameComponent(ref.Repository)
	safeTag := sanitizeFilenameComponent(ref.Tag)
	return fmt.Sprintf("%s_%s.tar.gz", safeImageName, safeTag)
}

// Dir returns the cache directory path
func (c *CacheManager) Dir() string {
	return c.dir
}
