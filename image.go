package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const sha256Prefix = "sha256:"

// authenticateClient authenticates with the registry and returns the client
func authenticateClient(ref ImageReference) (*RegistryClient, error) {
	client := NewRegistryClient()

	log.Printf("Authenticating with %s...\n", ref.Registry)
	if err := client.Authenticate(ref); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	log.Printf("Authenticated as: %s\n", client.GetAuthenticatedUser())

	return client, nil
}

// fetchManifest retrieves the manifest for the image
func fetchManifest(client *RegistryClient, ref ImageReference) (*ManifestV2, error) {
	log.Printf("Fetching manifest for %s:%s...\n", ref.Repository, ref.Tag)
	manifest, err := client.getManifest(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}
	return manifest, nil
}

// downloadImageConfig downloads and parses the image configuration
func downloadImageConfig(client *RegistryClient, ref ImageReference, manifest *ManifestV2, tempDir string) (*ImageConfig, string, error) {
	log.Println("Downloading image config...")
	configDigest := strings.TrimPrefix(manifest.Config.Digest, sha256Prefix)
	configPath := filepath.Join(tempDir, configDigest+".json")
	if err := client.DownloadBlob(ref, manifest.Config.Digest, configPath); err != nil {
		return nil, "", fmt.Errorf("failed to download config: %w", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", err
	}

	var imageConfig ImageConfig
	if err := json.Unmarshal(configData, &imageConfig); err != nil {
		return nil, "", err
	}

	return &imageConfig, configDigest, nil
}

// downloadAndProcessLayer downloads a single layer and creates its metadata files
func downloadAndProcessLayer(client *RegistryClient, ref ImageReference, layerDigestFull string, index int, totalLayers int, imageConfig *ImageConfig, tempDir string) (string, error) {
	log.Printf("Downloading layer %d/%d: %s\n", index+1, totalLayers, layerDigestFull[:19]+"...")
	layerDigest := strings.TrimPrefix(layerDigestFull, sha256Prefix)

	compressedPath := filepath.Join(tempDir, layerDigest+".tar.gz")
	if err := client.DownloadBlob(ref, layerDigestFull, compressedPath); err != nil {
		return "", fmt.Errorf("failed to download layer: %w", err)
	}

	diffID := strings.TrimPrefix(imageConfig.RootFS.DiffIDs[index], sha256Prefix)
	layerDir := filepath.Join(tempDir, diffID)
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return "", err
	}

	layerTarPath := filepath.Join(layerDir, "layer.tar")
	if err := decompressGzip(compressedPath, layerTarPath); err != nil {
		return "", fmt.Errorf("failed to decompress layer: %w", err)
	}

	if err := createLayerMetadata(layerDir, diffID, index, imageConfig); err != nil {
		return "", err
	}

	return diffID, nil
}

// createLayerMetadata creates VERSION and json files for a layer
func createLayerMetadata(layerDir, diffID string, index int, imageConfig *ImageConfig) error {
	if err := os.WriteFile(filepath.Join(layerDir, "VERSION"), []byte("1.0"), 0644); err != nil {
		return err
	}

	layerJSON := map[string]interface{}{
		"id":      diffID,
		"created": "0001-01-01T00:00:00Z",
	}
	if index > 0 {
		prevDiffID := strings.TrimPrefix(imageConfig.RootFS.DiffIDs[index-1], sha256Prefix)
		layerJSON["parent"] = prevDiffID
	}

	layerJSONBytes, _ := json.Marshal(layerJSON)
	return os.WriteFile(filepath.Join(layerDir, "json"), layerJSONBytes, 0644)
}

// downloadAllLayers downloads all layers and returns their diff IDs
func downloadAllLayers(client *RegistryClient, ref ImageReference, manifest *ManifestV2, imageConfig *ImageConfig, tempDir string) ([]string, error) {
	layerPaths := make([]string, len(manifest.Layers))

	for i, layer := range manifest.Layers {
		diffID, err := downloadAndProcessLayer(client, ref, layer.Digest, i, len(manifest.Layers), imageConfig, tempDir)
		if err != nil {
			return nil, err
		}
		layerPaths[i] = diffID
	}

	return layerPaths, nil
}

// createDockerManifest creates the manifest.json file for docker load
func createDockerManifest(ref ImageReference, configDigest string, layerPaths []string, tempDir string) error {
	repoTag := ref.Repository + ":" + ref.Tag
	if ref.Registry != "registry-1.docker.io" {
		repoTag = ref.Registry + "/" + repoTag
	}

	layers := make([]string, len(layerPaths))
	for i, p := range layerPaths {
		layers[i] = p + "/layer.tar"
	}

	manifestJSON := []map[string]interface{}{
		{
			"Config":   configDigest + ".json",
			"RepoTags": []string{repoTag},
			"Layers":   layers,
		},
	}

	manifestJSONBytes, _ := json.Marshal(manifestJSON)
	return os.WriteFile(filepath.Join(tempDir, "manifest.json"), manifestJSONBytes, 0644)
}

// createRepositoriesFile creates the repositories file for docker load
func createRepositoriesFile(ref ImageReference, layerPaths []string, tempDir string) error {
	imageName := filepath.Base(ref.Repository)
	topLayer := layerPaths[len(layerPaths)-1]

	repositories := map[string]map[string]string{
		imageName: {ref.Tag: topLayer},
	}

	reposBytes, _ := json.Marshal(repositories)
	return os.WriteFile(filepath.Join(tempDir, "repositories"), reposBytes, 0644)
}

// createOutputTar creates the final tar archive
func createOutputTar(ref ImageReference, tempDir, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	safeImageName := strings.ReplaceAll(ref.Repository, "/", "_")
	safePlatform := strings.ReplaceAll(ref.Platform.String(), "/", "_")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s_%s.tar.gz", safeImageName, ref.Tag, safePlatform))

	log.Println("Creating tar archive...")
	if err := createTar(tempDir, outputPath); err != nil {
		return "", fmt.Errorf("failed to create tar: %w", err)
	}

	log.Printf("Image saved to: %s\n", outputPath)
	return outputPath, nil
}

// DownloadImage downloads a Docker image and saves it as a tar file
// platform should be in format "os/architecture" (e.g., "linux/amd64", "linux/arm64")
func DownloadImage(imageRef string, outputDir string, platform string) (string, error) {
	ref := ParseImageReference(imageRef)
	ref.Platform = ParsePlatform(platform)

	client, err := authenticateClient(ref)
	if err != nil {
		return "", err
	}

	manifest, err := fetchManifest(client, ref)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "docker-image-*")
	if err != nil {
		return "", err
	}
	defer func(path string) {
		if err := os.RemoveAll(path); err != nil {
			log.Printf("Failed to remove temp dir: %s\n", err)
		}
	}(tempDir)

	imageConfig, configDigest, err := downloadImageConfig(client, ref, manifest, tempDir)
	if err != nil {
		return "", err
	}

	layerPaths, err := downloadAllLayers(client, ref, manifest, imageConfig, tempDir)
	if err != nil {
		return "", err
	}

	if err := createDockerManifest(ref, configDigest, layerPaths, tempDir); err != nil {
		return "", err
	}

	if err := createRepositoriesFile(ref, layerPaths, tempDir); err != nil {
		return "", err
	}

	return createOutputTar(ref, tempDir, outputDir)
}
