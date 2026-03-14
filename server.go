package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/singleflight"
)

//go:embed index.html logo.png
var staticFiles embed.FS

const contentTypeHeader = "Content-Type"

// Server represents the HTTP server for the Docker image service
type Server struct {
	addr          string
	cache         *CacheManager
	downloadGroup singleflight.Group
}

// NewServer creates a new server instance with a cache directory
func NewServer(addr string, cacheDir string, maxCacheAge time.Duration) *Server {
	cache, err := NewCacheManager(cacheDir, maxCacheAge)
	if err != nil {
		log.Fatalf("failed to initialize cache: %v", err)
	}

	return NewServerWithCache(addr, cache)
}

// NewServerWithCache creates a new server instance with a custom cache manager
func NewServerWithCache(addr string, cache *CacheManager) *Server {
	return &Server{addr: addr, cache: cache}
}

// Start starts the HTTP server and returns the *http.Server for shutdown control.
// It begins accepting connections immediately in a background goroutine.
func (s *Server) Start(ctx context.Context) (*http.Server, error) {
	// Start background cache cleanup
	go s.cache.StartCleanup(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.homeHandler)
	mux.HandleFunc("GET /health", s.healthHandler)
	mux.HandleFunc("GET /image", s.imageHandler)
	mux.HandleFunc("GET /logo.png", s.logoHandler)
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	log.Printf("Starting server on %s (cache: %s)\n", s.addr, s.cache.Dir())
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	return srv, nil
}

// healthHandler handles the /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprintln(w, "OK")
	if err != nil {
		log.Printf("Failed to write health response: %v\n", err)
	}
}

// homeHandler serves the main website at /
func (s *Server) homeHandler(w http.ResponseWriter, _ *http.Request) {
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
func (s *Server) logoHandler(w http.ResponseWriter, _ *http.Request) {
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

	cachePath := s.cache.GetCachePath(imageName)

	if _, err := os.Stat(cachePath); err == nil {
		log.Printf("Serving cached image: %s\n", imageName)
		s.serveImageFile(w, r, cachePath, imageName)
		return
	}

	log.Printf("Downloading image: %s\n", imageName)
	result, err, _ := s.downloadGroup.Do(imageName, func() (interface{}, error) {
		return DownloadImage(imageName, s.cache.Dir())
	})
	if err != nil {
		log.Printf("Failed to download image %s: %v\n", imageName, err)
		errorsTotalMetric.Inc()
		writeJSONError(w, fmt.Sprintf("failed to download image: %v", err), http.StatusInternalServerError)
		return
	}
	imagePath := result.(string)

	s.serveImageFile(w, r, imagePath, imageName)
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

	if err := os.Chtimes(imagePath, time.Now(), time.Now()); err != nil {
		log.Printf("Failed to update access time for %s: %v\n", imagePath, err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to stat image file: %v\n", err)
		errorsTotalMetric.Inc()
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	filename := s.cache.GetCacheFilename(imageName)

	w.Header().Set(contentTypeHeader, "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)

	log.Printf("Served image: %s (%s)\n", imageName, humanizeBytes(fileInfo.Size()))
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

// humanizeBytes converts bytes to a human-readable format
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
