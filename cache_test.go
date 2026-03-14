package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPerformCleanup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-cleanup-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	maxAge := 2 * time.Hour

	// Create a new file (should NOT be removed)
	newFile := filepath.Join(tempDir, "new_file.tar.gz")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an old file (should be removed)
	oldFile := filepath.Join(tempDir, "old_file.tar.gz")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	// Manually set the modification time to be older than maxAge
	oldTime := time.Now().Add(-maxAge - time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	cache, _ := NewCacheManager(tempDir, maxAge)
	cache.PerformCleanup()

	// Check if the new file still exists
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Errorf("new file was incorrectly removed")
	}

	// Check if the old file was removed
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old file was not removed")
	}
}

func TestGetCacheFilename(t *testing.T) {
	cache, _ := NewCacheManager("", 1*time.Hour)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("failed to remove temporary directory: %v", err)
		}
	}(cache.Dir())

	tests := []struct {
		imageName string
		expected  string
	}{
		{
			imageName: "alpine:latest",
			expected:  "library_alpine_latest.tar.gz",
		},
		{
			imageName: "library/ubuntu:20.04",
			expected:  "library_ubuntu_20.04.tar.gz",
		},
		{
			imageName: "ghcr.io/username/repo:v1.2.3",
			expected:  "username_repo_v1.2.3.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.imageName, func(t *testing.T) {
			got := cache.GetCacheFilename(tt.imageName)
			if got != tt.expected {
				t.Errorf("GetCacheFilename(%q) = %q, want %q", tt.imageName, got, tt.expected)
			}
		})
	}
}
