package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	registryPattern   = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*(:[0-9]+)?$`)
	repositoryPattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`)
	tagPattern        = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)
	digestPattern     = regexp.MustCompile(`^[a-z0-9]+:[a-f0-9]+$`)
	imageNamePattern  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/:]*$`)
)

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

	if host == "localhost" || host == "127.0.0.1" {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	validIP, isPrivate, _ := parseIPRange(host)
	if validIP && isPrivate {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	if host == "169.254.169.254" {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	if !strings.Contains(host, ".") && isNumeric(host) {
		return fmt.Errorf("registry hostname not allowed: %s", registry)
	}

	parts := strings.Split(host, ".")
	if len(parts) == 4 && allNumeric(parts) {
		for _, part := range parts {
			if len(part) > 1 && part[0] == '0' {
				return fmt.Errorf("registry hostname not allowed: %s", registry)
			}
		}
	}

	for _, part := range parts {
		if strings.HasPrefix(part, "0x") || strings.HasPrefix(part, "0X") {
			return fmt.Errorf("registry hostname not allowed: %s", registry)
		}
	}

	if len(parts) == 4 && allNumeric(parts) {
		for _, part := range parts {
			if len(part) > 1 && part[0] == '0' {
				return fmt.Errorf("registry hostname not allowed: %s", registry)
			}
		}
	}

	return nil
}

func parseIPRange(host string) (bool, bool, bool) {
	isPrivate := false
	isLoopback := false
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false, isPrivate, isLoopback
	}
	for _, p := range parts {
		if !isNumeric(p) {
			return false, isPrivate, isLoopback
		}
	}
	a := parts[0]
	b := parts[1]
	c := parts[2]
	d := parts[3]

	isLoopback = host == "127.0.0.1"

	isPrivate = (a == "10") ||
		(a == "192" && b == "168") ||
		(a == "172" && b >= "16" && b <= "31") ||
		(a == "169" && b == "254" && c == "169" && d == "254") ||
		host == "0.0.0.0"

	return true, isPrivate, isLoopback
}

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

func allNumeric(parts []string) bool {
	for _, p := range parts {
		if !isNumeric(p) {
			return false
		}
	}
	return true
}

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

func validateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid tag: %s", tag)
	}
	return nil
}

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

func buildRegistryURL(registry, pathFormat string, args ...interface{}) (string, error) {
	if err := validateRegistry(registry); err != nil {
		return "", err
	}
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

	ref := ParseImageReference(imageName)
	if err := ValidateImageReference(ref); err != nil {
		return "", err
	}

	return imageName, nil
}

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
