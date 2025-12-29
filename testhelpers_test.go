package main

import (
	"os"
	"testing"
)

func cleanupTempDir(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("failed to remove temp dir: %v", err)
	}
}
