package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const bearerPrefix = "Bearer "
const responseBodyStr = "response body"

// ImageReference represents a parsed Docker image reference
type ImageReference struct {
	Registry   string
	Repository string
	Tag        string
}

// RegistryClient handles communication with Docker registries
type RegistryClient struct {
	httpClient *http.Client
	token      string
	username   string // Track authenticated user for logging
}

// ManifestV2 represents a Docker manifest schema v2
type ManifestV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

// ManifestList represents a multi-platform manifest list
type ManifestList struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Manifests     []struct {
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// ImageConfig represents the image configuration
type ImageConfig struct {
	Architecture string    `json:"architecture"`
	Created      time.Time `json:"created"`
	OS           string    `json:"os"`
	Config       struct {
		Env        []string `json:"Env"`
		Cmd        []string `json:"Cmd"`
		WorkingDir string   `json:"WorkingDir"`
	} `json:"config"`
	RootFS struct {
		Type    string   `json:"type"`
		DiffIDs []string `json:"diff_ids"`
	} `json:"rootfs"`
}

// ParseImageReference parses an image reference string
func ParseImageReference(ref string) ImageReference {
	result := ImageReference{
		Registry: "registry-1.docker.io",
		Tag:      "latest",
	}

	if idx := strings.LastIndex(ref, ":"); idx != -1 && !strings.Contains(ref[idx:], "/") {
		result.Tag = ref[idx+1:]
		ref = ref[:idx]
	}

	parts := strings.Split(ref, "/")
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		result.Registry = normalizeRegistry(parts[0])
		ref = strings.Join(parts[1:], "/")
	}

	if result.Registry == "registry-1.docker.io" && !strings.Contains(ref, "/") {
		ref = "library/" + ref
	}

	result.Repository = ref
	return result
}

// NewRegistryClient creates a new registry client
func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

// Authenticate obtains a token for the given image
func (c *RegistryClient) Authenticate(ref ImageReference) error {
	creds, hasCredentials := GetCredentials(ref.Registry)
	if hasCredentials {
		c.username = creds.Username
	} else {
		c.username = "anonymous"
	}

	url := fmt.Sprintf("https://%s/v2/", ref.Registry)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer closeWithLog(resp.Body, responseBodyStr)

	if resp.StatusCode == http.StatusOK {
		return nil // No auth required
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("no WWW-Authenticate header")
	}

	realm, service, scope := parseAuthHeader(authHeader, ref.Repository)

	tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)

	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return err
	}

	if hasCredentials {
		auth := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer closeWithLog(resp.Body, responseBodyStr)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %d - %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	c.token = tokenResp.Token
	if c.token == "" {
		c.token = tokenResp.AccessToken
	}

	return nil
}

// GetAuthenticatedUser returns the username used for authentication
func (c *RegistryClient) GetAuthenticatedUser() string {
	return c.username
}

func parseAuthHeader(header, repo string) (realm, service, scope string) {
	header = strings.TrimPrefix(header, bearerPrefix)
	parts := strings.Split(header, ",")

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.Trim(kv[1], "\"")

		switch key {
		case "realm":
			realm = value
		case "service":
			service = value
		}
	}

	scope = fmt.Sprintf("repository:%s:pull", repo)
	return
}

const manifestAcceptHeader = "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"

// isManifestList checks if the content type indicates a manifest list or image index
func isManifestList(contentType string) bool {
	return strings.Contains(contentType, "manifest.list") || strings.Contains(contentType, "image.index")
}

// selectManifestDigest selects the best manifest from a manifest list, preferring linux/amd64
func (c *RegistryClient) selectManifestDigest(ref ImageReference, list *ManifestList) (*ManifestV2, error) {
	for _, m := range list.Manifests {
		if m.Platform.OS == "linux" && m.Platform.Architecture == "amd64" {
			return c.getManifestByDigest(ref, m.Digest)
		}
	}
	if len(list.Manifests) > 0 {
		return c.getManifestByDigest(ref, list.Manifests[0].Digest)
	}
	return nil, fmt.Errorf("no suitable manifest found")
}

// parseManifestResponse parses the manifest response body based on content type
func (c *RegistryClient) parseManifestResponse(ref ImageReference, contentType string, body []byte) (*ManifestV2, error) {
	if isManifestList(contentType) {
		var list ManifestList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, err
		}
		return c.selectManifestDigest(ref, &list)
	}

	var manifest ManifestV2
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// fetchManifestResponse fetches the raw manifest response from the registry
func (c *RegistryClient) fetchManifestResponse(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", manifestAcceptHeader)
	if c.token != "" {
		req.Header.Set("Authorization", bearerPrefix+c.token)
	}

	return c.httpClient.Do(req)
}

// getManifest retrieves the image manifest
func (c *RegistryClient) getManifest(ref ImageReference) (*ManifestV2, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repository, ref.Tag)

	resp, err := c.fetchManifestResponse(url)
	if err != nil {
		return nil, err
	}
	defer closeWithLog(resp.Body, responseBodyStr)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get manifest: %d - %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return c.parseManifestResponse(ref, contentType, body)
}

func (c *RegistryClient) getManifestByDigest(ref ImageReference, digest string) (*ManifestV2, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json")
	if c.token != "" {
		req.Header.Set("Authorization", bearerPrefix+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeWithLog(resp.Body, responseBodyStr)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get manifest by digest: %d", resp.StatusCode)
	}

	var manifest ManifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// DownloadBlob downloads a blob to a file
func (c *RegistryClient) DownloadBlob(ref ImageReference, digest, destPath string) error {
	url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", ref.Registry, ref.Repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if c.token != "" {
		req.Header.Set("Authorization", bearerPrefix+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer closeWithLog(resp.Body, responseBodyStr)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download blob: %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer closeWithLog(file, "blob file")

	_, err = io.Copy(file, resp.Body)
	return err
}
