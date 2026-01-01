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
)

//go:embed index.html logo.png
var staticFiles embed.FS

const contentTypeHeader = "Content-Type"

// Server represents the HTTP server for the Docker image service
type Server struct {
	addr     string
	cacheDir string
}

// NewServer creates a new server instance with a cache directory
func NewServer(addr string, cacheDir string) *Server {
	if cacheDir == "" {
		tmpDir, err := os.MkdirTemp("", "docker-image-cache-*")
		if err != nil {
			log.Fatalf("failed to create temporary cache directory: %v", err)
		}
		cacheDir = tmpDir
	} else if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("failed to create cache directory: %v", err)
	}

	return NewServerWithCache(addr, cacheDir)
}

// NewServerWithCache creates a new server instance with a custom cache directory
func NewServerWithCache(addr, cacheDir string) *Server {
	return &Server{addr: addr, cacheDir: cacheDir}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	if err := os.MkdirAll(s.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.homeHandler)
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /image", s.imageHandler)
	mux.HandleFunc("GET /logo.png", s.logoHandler)
	mux.Handle("GET /metrics", promhttp.Handler())

	log.Printf("Starting server on %s (cache: %s)\n", s.addr, s.cacheDir)
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

	cacheFilename := s.getCacheFilename(imageName)
	cachePath := filepath.Join(s.cacheDir, cacheFilename)

	if _, err := os.Stat(cachePath); err == nil {
		log.Printf("Serving cached image: %s\n", imageName)
		s.serveImageFile(w, r, cachePath, imageName)
		return
	}

	log.Printf("Downloading image: %s\n", imageName)
	imagePath, err := DownloadImage(imageName, s.cacheDir)
	if err != nil {
		log.Printf("Failed to download image %s: %v\n", imageName, err)
		errorsTotalMetric.Inc()
		writeJSONError(w, fmt.Sprintf("failed to download image: %v", err), http.StatusInternalServerError)
		return
	}

	s.serveImageFile(w, r, imagePath, imageName)
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
func (s *Server) getCacheFilename(imageName string) string {
	ref := ParseImageReference(imageName)
	safeImageName := strings.ReplaceAll(ref.Repository, "/", "_")
	return fmt.Sprintf("%s_%s.tar.gz", safeImageName, ref.Tag)
}

// serveImageFile streams an image tar file to the response with Range request support
func (s *Server) serveImageFile(w http.ResponseWriter, r *http.Request, imagePath, imageName string) {
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

	filename := s.getCacheFilename(imageName)

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
