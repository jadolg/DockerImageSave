package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDockerManifest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-manifest-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	ref := ImageReference{
		Registry:   "registry-1.docker.io",
		Repository: "library/alpine",
		Tag:        "latest",
	}
	configDigest := "abc123def456"
	layerPaths := []string{"layer1", "layer2"}

	if err := createDockerManifest(ref, configDigest, layerPaths, tempDir); err != nil {
		t.Fatalf("createDockerManifest failed: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	var manifests []map[string]interface{}
	if err := json.Unmarshal(data, &manifests); err != nil {
		t.Fatalf("failed to parse manifest.json: %v", err)
	}

	if len(manifests) != 1 {
		t.Errorf("expected 1 manifest entry, got %d", len(manifests))
	}

	m := manifests[0]
	if m["Config"] != "abc123def456.json" {
		t.Errorf("expected Config 'abc123def456.json', got '%v'", m["Config"])
	}

	repoTags := m["RepoTags"].([]interface{})
	if len(repoTags) != 1 || repoTags[0] != "library/alpine:latest" {
		t.Errorf("unexpected RepoTags: %v", repoTags)
	}

	layers := m["Layers"].([]interface{})
	if len(layers) != 2 {
		t.Errorf("expected 2 layers, got %d", len(layers))
	}
	if layers[0] != "layer1/layer.tar" {
		t.Errorf("expected first layer 'layer1/layer.tar', got '%v'", layers[0])
	}
}

func TestCreateDockerManifest_CustomRegistry(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-manifest-custom-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	ref := ImageReference{
		Registry:   "gcr.io",
		Repository: "my-project/my-image",
		Tag:        "v1.0",
	}
	configDigest := "xyz789"
	layerPaths := []string{"layer1"}

	if err := createDockerManifest(ref, configDigest, layerPaths, tempDir); err != nil {
		t.Fatalf("createDockerManifest failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	var manifests []map[string]interface{}
	if err := json.Unmarshal(data, &manifests); err != nil {
		t.Fatal(err)
	}

	repoTags := manifests[0]["RepoTags"].([]interface{})
	expected := "gcr.io/my-project/my-image:v1.0"
	if repoTags[0] != expected {
		t.Errorf("expected RepoTag '%s', got '%v'", expected, repoTags[0])
	}
}

func TestCreateRepositoriesFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-repos-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	ref := ImageReference{
		Registry:   "registry-1.docker.io",
		Repository: "library/alpine",
		Tag:        "3.18",
	}
	layerPaths := []string{"layer1", "layer2", "layer3"}

	if err := createRepositoriesFile(ref, layerPaths, tempDir); err != nil {
		t.Fatalf("createRepositoriesFile failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, "repositories"))
	if err != nil {
		t.Fatal(err)
	}

	var repos map[string]map[string]string
	if err := json.Unmarshal(data, &repos); err != nil {
		t.Fatalf("failed to parse repositories: %v", err)
	}

	if repos["alpine"]["3.18"] != "layer3" {
		t.Errorf("expected alpine:3.18 -> layer3, got %v", repos)
	}
}

func TestCreateLayerMetadata(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-layer-meta-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	layerDir := filepath.Join(tempDir, "layer1")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	imageConfig := &ImageConfig{}
	imageConfig.RootFS.DiffIDs = []string{"sha256:abc", "sha256:def"}

	if err := createLayerMetadata(layerDir, "abc123", 0, imageConfig); err != nil {
		t.Fatalf("createLayerMetadata failed: %v", err)
	}

	version, err := os.ReadFile(filepath.Join(layerDir, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if string(version) != "1.0" {
		t.Errorf("expected VERSION '1.0', got '%s'", version)
	}

	jsonData, err := os.ReadFile(filepath.Join(layerDir, "json"))
	if err != nil {
		t.Fatal(err)
	}

	var layerJSON map[string]interface{}
	if err := json.Unmarshal(jsonData, &layerJSON); err != nil {
		t.Fatal(err)
	}

	if layerJSON["id"] != "abc123" {
		t.Errorf("expected id 'abc123', got '%v'", layerJSON["id"])
	}
	if _, hasParent := layerJSON["parent"]; hasParent {
		t.Error("first layer should not have parent")
	}
}

func TestCreateLayerMetadata_WithParent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-layer-meta-parent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	layerDir := filepath.Join(tempDir, "layer2")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	imageConfig := &ImageConfig{}
	imageConfig.RootFS.DiffIDs = []string{"sha256:parenthash", "sha256:currenthash"}

	if err := createLayerMetadata(layerDir, "currenthash", 1, imageConfig); err != nil {
		t.Fatalf("createLayerMetadata failed: %v", err)
	}

	jsonData, err := os.ReadFile(filepath.Join(layerDir, "json"))
	if err != nil {
		t.Fatal(err)
	}

	var layerJSON map[string]interface{}
	if err := json.Unmarshal(jsonData, &layerJSON); err != nil {
		t.Fatal(err)
	}

	if layerJSON["parent"] != "parenthash" {
		t.Errorf("expected parent 'parenthash', got '%v'", layerJSON["parent"])
	}
}

func TestDownloadImage_PublicImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	outputDir, err := os.MkdirTemp("", "test-download-public-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, outputDir)

	imagePath, err := DownloadImage("alpine:latest", outputDir, "")
	if err != nil {
		t.Fatalf("DownloadImage failed: %v", err)
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Errorf("expected image file to exist at %s", imagePath)
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("expected non-zero file size")
	}
}

func TestDownloadImage_WithAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	outputDir, err := os.MkdirTemp("", "test-download-auth-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, outputDir)

	imagePath, err := DownloadImage("busybox:latest", outputDir, "")
	if err != nil {
		t.Fatalf("DownloadImage with auth failed: %v", err)
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Errorf("expected image file to exist at %s", imagePath)
	}
}

func TestDownloadImage_NonExistentImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	outputDir, err := os.MkdirTemp("", "test-download-nonexistent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, outputDir)

	_, err = DownloadImage("thisimagedoesnotexist12345:nonexistenttag", outputDir, "")
	if err == nil {
		t.Error("expected error for non-existent image")
	}
}

func TestDownloadImage_WithPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name     string
		image    string
		platform string
	}{
		{
			name:     "linux/amd64",
			image:    "alpine:latest",
			platform: "linux/amd64",
		},
		{
			name:     "linux/arm64",
			image:    "alpine:latest",
			platform: "linux/arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir, err := os.MkdirTemp("", "test-download-platform-*")
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupTempDir(t, outputDir)

			imagePath, err := DownloadImage(tt.image, outputDir, tt.platform)
			if err != nil {
				t.Fatalf("DownloadImage with platform %s failed: %v", tt.platform, err)
			}

			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				t.Errorf("expected image file to exist at %s", imagePath)
			}

			info, err := os.Stat(imagePath)
			if err != nil {
				t.Fatal(err)
			}
			if info.Size() == 0 {
				t.Error("expected non-zero file size")
			}

			// Verify filename includes platform
			platformParts := strings.Split(tt.platform, "/")
			if len(platformParts) != 2 {
				t.Fatalf("invalid platform format %q, expected <os>/<arch>", tt.platform)
			}
			expectedSuffix := "_" + platformParts[0] + "_" + platformParts[1] + ".tar.gz"
			if !strings.HasSuffix(imagePath, expectedSuffix) {
				t.Errorf("expected filename to end with '%s', got '%s'", expectedSuffix, imagePath)
			}
		})
	}
}

func TestDownloadImage_UnsupportedPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	outputDir, err := os.MkdirTemp("", "test-download-unsupported-platform-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, outputDir)

	// Try to download with a platform that doesn't exist for this image
	_, err = DownloadImage("alpine:latest", outputDir, "windows/arm64")
	if err == nil {
		t.Error("expected error for unsupported platform")
	}
}

func TestCreateOutputTar_IncludesPlatformInFilename(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-output-tar-platform-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	outputDir, err := os.MkdirTemp("", "test-output-dir-platform-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, outputDir)

	// Create a minimal file structure for tar
	if err := os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		ref      ImageReference
		expected string
	}{
		{
			name: "linux_amd64 platform",
			ref: ImageReference{
				Registry:   "registry-1.docker.io",
				Repository: "library/alpine",
				Tag:        "latest",
				Platform:   Platform{OS: "linux", Architecture: "amd64"},
			},
			expected: "library_alpine_latest_linux_amd64.tar.gz",
		},
		{
			name: "linux_arm64 platform",
			ref: ImageReference{
				Registry:   "registry-1.docker.io",
				Repository: "library/alpine",
				Tag:        "3.18",
				Platform:   Platform{OS: "linux", Architecture: "arm64"},
			},
			expected: "library_alpine_3.18_linux_arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testOutputDir, err := os.MkdirTemp("", "test-output-*")
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupTempDir(t, testOutputDir)

			outputPath, err := createOutputTar(tt.ref, tempDir, testOutputDir)
			if err != nil {
				t.Fatalf("createOutputTar failed: %v", err)
			}

			expectedPath := filepath.Join(testOutputDir, tt.expected)
			if outputPath != expectedPath {
				t.Errorf("expected path '%s', got '%s'", expectedPath, outputPath)
			}
		})
	}
}
