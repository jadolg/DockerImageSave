package main

import "strings"

// sanitizePathComponent removes dangerous characters from a string so it can be safely
// used as a single path component (e.g., filename segment). This is used for both cache
// filenames and output tar filenames to prevent path traversal.
func sanitizePathComponent(s string) string {
	// Replace path separators with underscores
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	// Remove path traversal sequences
	s = strings.ReplaceAll(s, "..", "")
	// Remove any remaining dots at the start (hidden files)
	s = strings.TrimLeft(s, ".")
	return s
}
