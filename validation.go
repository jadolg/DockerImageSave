package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Validation patterns for Docker registry components
var (
	registryPattern   = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*(:[0-9]+)?$`)
	repositoryPattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`)
	tagPattern        = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)
	digestPattern     = regexp.MustCompile(`^[a-z0-9]+:[a-f0-9]+$`)
	imageNamePattern  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/:]*$`)
)

// validateRegistry validates and sanitizes a registry hostname
func validateRegistry(registry string) error {
	if registry == "" {
		return fmt.Errorf("registry cannot be empty")
	}
	if len(registry) > 253 {
		return fmt.Errorf("registry hostname too long")
	}
	if !registryPattern.MatchString(registry) {
		return fmt.Errorf("invalid registry hostname: %s", registry)
	}
	lower := strings.ToLower(registry)
	host := strings.Split(lower, ":")[0]

	// Block localhost variants
	if host == "localhost" || host == "127.0.0.1" {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	// Block private IP ranges
	if strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "172.") {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	// Block special addresses
	if host == "0.0.0.0" || host == "169.254.169.254" {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	// Block decimal IP representation (e.g., 2130706433 = 127.0.0.1)
	// A valid registry should have at least one dot for a TLD
	if !strings.Contains(host, ".") && isNumeric(host) {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	// Block zero-padded IPs (e.g., 127.0.0.01)
	parts := strings.Split(host, ".")
	if len(parts) == 4 && allNumeric(parts) {
		for _, part := range parts {
			if len(part) > 1 && part[0] == '0' {
				return fmt.Errorf("registry hostname not allowed: %s", registry)
			}
		}
	}

	// Block hex notation (e.g., 0x7f.0.0.1)
	for _, part := range parts {
		if strings.HasPrefix(part, "0x") || strings.HasPrefix(part, "0X") {
			return fmt.Errorf("registry hostname not allowed: %s", registry)
		}
	}

	// Block octal notation (e.g., 0177.0.0.1) - numbers starting with 0 that aren't just "0"
	if len(parts) == 4 && allNumeric(parts) {
		for _, part := range parts {
			if len(part) > 1 && part[0] == '0' {
				return fmt.Errorf("registry hostname not allowed: %s", registry)
			}
		}
	}

	return nil
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// allNumeric checks if all strings in a slice are numeric
func allNumeric(parts []string) bool {
	for _, p := range parts {
		if !isNumeric(p) {
			return false
		}
	}
	return true
}

// validateRepository validates a repository name
func validateRepository(repository string) error {
	if repository == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	if len(repository) > 256 {
		return fmt.Errorf("repository name too long")
	}
	if !repositoryPattern.MatchString(repository) {
		return fmt.Errorf("invalid repository name: %s", repository)
	}
	return nil
}

// validateTag validates a tag name
func validateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid tag: %s", tag)
	}
	return nil
}

// validateDigest validates a digest string
func validateDigest(digest string) error {
	if digest == "" {
		return fmt.Errorf("digest cannot be empty")
	}
	if len(digest) > 256 {
		return fmt.Errorf("digest too long")
	}
	if !digestPattern.MatchString(digest) {
		return fmt.Errorf("invalid digest format: %s", digest)
	}
	return nil
}

// ValidateImageReference validates all components of an image reference
func ValidateImageReference(ref ImageReference) error {
	if err := validateRegistry(ref.Registry); err != nil {
		return err
	}
	if err := validateRepository(ref.Repository); err != nil {
		return err
	}
	if err := validateTag(ref.Tag); err != nil {
		return err
	}
	return nil
}

// buildRegistryURL safely constructs a registry URL with proper escaping
func buildRegistryURL(registry, pathFormat string, args ...interface{}) (string, error) {
	if err := validateRegistry(registry); err != nil {
		return "", err
	}
	// Escape path components
	escapedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			escapedArgs[i] = url.PathEscape(s)
		} else {
			escapedArgs[i] = arg
		}
	}
	path := fmt.Sprintf(pathFormat, escapedArgs...)
	return fmt.Sprintf("https://%s%s", registry, path), nil
}

// sanitizeImageName validates and sanitizes a Docker image name from user input
func sanitizeImageName(imageName string) (string, error) {
	imageName = strings.TrimSpace(imageName)

	if imageName == "" {
		return "", fmt.Errorf("image name cannot be empty")
	}

	if len(imageName) > 256 {
		return "", fmt.Errorf("image name too long (max 256 characters)")
	}

	if strings.Contains(imageName, "..") {
		return "", fmt.Errorf("invalid characters in image name")
	}

	if !imageNamePattern.MatchString(imageName) {
		return "", fmt.Errorf("image name contains invalid characters")
	}

	// Parse and validate the image reference components
	ref := ParseImageReference(imageName)
	if err := ValidateImageReference(ref); err != nil {
		return "", err
	}

	return imageName, nil
}

// sanitizeFilenameComponent normalizes a string so it is safe to use as a single path component.
// It removes path separators and parent directory references that could lead to path traversal.
func sanitizeFilenameComponent(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "_")
	s = strings.TrimSpace(s)
	if s == "" {
		s = "unknown"
	}
	return s
}
