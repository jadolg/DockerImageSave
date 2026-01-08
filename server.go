package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/singleflight"
)

//go:embed index.html logo.png
var staticFiles embed.FS

const contentTypeHeader = "Content-Type"

// Server represents the HTTP server for the Docker image service
type Server struct {
	addr          string
	cacheDir      string
	downloadGroup singleflight.Group
	auth          *AuthMiddleware
}

// NewServer creates a new server instance with a cache directory
func NewServer(addr string, cacheDir string) *Server {
	return NewServerWithConfig(addr, cacheDir, nil)
}

// NewServerWithConfig creates a new server instance with optional auth configuration
func NewServerWithConfig(addr string, cacheDir string, authConfig *AuthConfig) *Server {
	if cacheDir == "" {
		tmpDir, err := os.MkdirTemp("", "docker-image-cache-*")
		if err != nil {
			log.Fatalf("failed to create temporary cache directory: %v", err)
		}
		cacheDir = tmpDir
	} else if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("failed to create cache directory: %v", err)
	}

	return &Server{
		addr:     addr,
		cacheDir: cacheDir,
		auth:     NewAuthMiddleware(authConfig),
	}
}

// NewServerWithCache creates a new server instance with a custom cache directory
func NewServerWithCache(addr, cacheDir string) *Server {
	return &Server{addr: addr, cacheDir: cacheDir, auth: NewAuthMiddleware(nil)}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	if err := os.MkdirAll(s.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	mux := http.NewServeMux()

	// Public endpoints (no auth required)
	mux.HandleFunc("GET /{$}", s.homeHandler)
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /logo.png", s.logoHandler)
	mux.Handle("GET /metrics", promhttp.Handler())

	// Protected endpoints (auth required when enabled)
	mux.HandleFunc("GET /image", s.auth.WrapFunc(s.imageHandler))

	if s.auth.IsEnabled() {
		log.Printf("Starting server on %s (cache: %s, auth: enabled)\n", s.addr, s.cacheDir)
	} else {
		log.Printf("Starting server on %s (cache: %s)\n", s.addr, s.cacheDir)
	}
	return http.ListenAndServe(s.addr, mux)
}

// healthHandler handles the /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintln(w, "OK")
	if err != nil {
		log.Printf("Failed to write health response: %v\n", err)
	}
}

// homeHandler serves the main website at /
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set(contentTypeHeader, "text/html; charset=utf-8")
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Failed to write home page response: %v\n", err)
	}
}

// logoHandler serves the logo.png file
func (s *Server) logoHandler(w http.ResponseWriter, r *http.Request) {
	data, err := staticFiles.ReadFile("logo.png")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set(contentTypeHeader, "image/png")
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Failed to write logo response: %v\n", err)
	}
}

// imageHandler handles the /image endpoint
func (s *Server) imageHandler(w http.ResponseWriter, r *http.Request) {

	imageName := r.URL.Query().Get("name")
	if imageName == "" {
		writeJSONError(w, "missing required 'name' query parameter", http.StatusBadRequest)
		return
	}

	imageName, err := sanitizeImageName(imageName)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("invalid image name: %v", err), http.StatusBadRequest)
		return
	}

	// Parse platform parameter (e.g., "linux/amd64", "linux/arm64")
	// URL-encoded slashes (%2F) are automatically decoded by Go's URL parser
	platform := r.URL.Query().Get("platform")
	if platform != "" {
		// Sanitize and validate platform - returns a safe reconstructed value
		sanitized, err := sanitizePlatform(platform)
		if err != nil {
			writeJSONError(w, fmt.Sprintf("invalid platform: %v", err), http.StatusBadRequest)
			return
		}
		platform = sanitized
	} else {
		// Normalize empty platform to default to ensure consistent cache keys
		// This prevents duplicate downloads when one request omits platform
		// and another explicitly specifies "linux/amd64"
		platform = DefaultPlatform().String()
	}

	// Create a unique cache key combining image name and platform
	cacheKey := imageName + ":" + platform
	cacheFilename := s.getCacheFilename(imageName, platform)
	cachePath := filepath.Join(s.cacheDir, cacheFilename)

	// Validate that the cache path stays within the cache directory (prevent path traversal)
	if err := validatePathContainment(s.cacheDir, cachePath); err != nil {
		log.Printf("Security: path traversal attempt detected for image %s: %v\n", imageName, err)
		writeJSONError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(cachePath); err == nil {
		log.Printf("Serving cached image: %s (platform: %s)\n", imageName, platform)
		s.serveImageFile(w, r, cachePath, imageName, platform)
		return
	}

	log.Printf("Downloading image: %s (platform: %s)\n", imageName, platform)
	result, err, _ := s.downloadGroup.Do(cacheKey, func() (interface{}, error) {
		return DownloadImage(imageName, s.cacheDir, platform)
	})
	if err != nil {
		log.Printf("Failed to download image %s: %v\n", imageName, err)
		errorsTotalMetric.Inc()
		writeJSONError(w, fmt.Sprintf("failed to download image: %v", err), http.StatusInternalServerError)
		return
	}
	imagePath := result.(string)

	s.serveImageFile(w, r, imagePath, imageName, platform)
}

// validatePlatform validates the platform string format and returns a sanitized version
// This prevents path traversal attacks by reconstructing the platform from validated components
func sanitizePlatform(platform string) (string, error) {
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("platform must be in format 'os/architecture' (e.g., 'linux/amd64')")
	}
	osName := parts[0]
	arch := parts[1]

	// Validate OS against whitelist
	validOS := map[string]bool{"linux": true, "windows": true, "darwin": true}
	if !validOS[osName] {
		return "", fmt.Errorf("unsupported OS '%s', valid options: linux, windows, darwin", osName)
	}

	// Validate architecture against whitelist
	validArch := map[string]bool{"amd64": true, "arm64": true, "arm": true, "386": true, "ppc64le": true, "s390x": true, "riscv64": true}
	if !validArch[arch] {
		return "", fmt.Errorf("unsupported architecture '%s', valid options: amd64, arm64, arm, 386, ppc64le, s390x, riscv64", arch)
	}

	// Return reconstructed platform from validated components (not user input)
	// This ensures path safety by only using known-good values
	return osName + "/" + arch, nil
}

var imageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/:]*$`)

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

	return imageName, nil
}

// getCacheFilename generates a safe filename for caching
// This must match the filename format used by createOutputTar in image.go
func (s *Server) getCacheFilename(imageName string, platform string) string {
	ref := ParseImageReference(imageName)
	ref.Platform = ParsePlatform(platform)
	safeImageName := sanitizePathComponent(ref.Repository)
	safeTag := sanitizePathComponent(ref.Tag)
	safePlatform := sanitizePathComponent(ref.Platform.String())
	return fmt.Sprintf("%s_%s_%s.tar.gz", safeImageName, safeTag, safePlatform)
}

// validatePathContainment ensures the final path stays within the base directory
func validatePathContainment(basePath, fullPath string) error {
	// Clean both paths for comparison
	cleanBase := filepath.Clean(basePath)
	cleanFull := filepath.Clean(fullPath)

	// Ensure the full path starts with the base path
	if !strings.HasPrefix(cleanFull, cleanBase+string(filepath.Separator)) && cleanFull != cleanBase {
		return fmt.Errorf("path traversal detected: path escapes base directory")
	}
	return nil
}

// serveImageFile streams an image tar file to the response with Range request support
func (s *Server) serveImageFile(w http.ResponseWriter, r *http.Request, imagePath, imageName string, platform string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Failed to open image file: %v\n", err)
		errorsTotalMetric.Inc()
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Failed to close image file: %v\n", err)
		}
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to stat image file: %v\n", err)
		errorsTotalMetric.Inc()
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	filename := s.getCacheFilename(imageName, platform)

	w.Header().Set(contentTypeHeader, "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)

	log.Printf("Served image: %s (%d bytes total)\n", imageName, fileInfo.Size())
	pullsCountMetric.Inc()
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set(contentTypeHeader, "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(map[string]string{"error": message})
	if err != nil {
		errorsTotalMetric.Inc()
		log.Printf("Failed to write JSON error response: %v\n", err)
	}
}
