package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

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
		log.WithError(err).Fatal("Failed to initialize cache")
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
	mux.HandleFunc("GET /platforms", s.platformsHandler)
	mux.HandleFunc("GET /logo.png", s.logoHandler)
	mux.Handle("GET /metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	log.WithFields(log.Fields{
		"addr":      s.addr,
		"cache_dir": s.cache.Dir(),
	}).Info("Starting server")
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Server error")
		}
	}()

	return srv, nil
}

// healthHandler handles the /health endpoint
func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprintln(w, "OK")
	if err != nil {
		log.WithError(err).Warn("Failed to write health response")
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
		log.WithError(err).Warn("Failed to write home page response")
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
		log.WithError(err).Warn("Failed to write logo response")
	}
}

// imageHandler handles the /image endpoint
func (s *Server) imageHandler(w http.ResponseWriter, r *http.Request) {
	imageName, ok := extractImageName(w, r)
	if !ok {
		return
	}

	platform, ok := platformFromRequest(w, r)
	if !ok {
		return
	}

	cachePath := s.cache.GetCachePath(imageName, platform)

	if _, err := os.Stat(cachePath); err == nil {
		log.WithFields(log.Fields{
			"image":    imageName,
			"platform": platform,
		}).Info("Serving cached image")
		s.serveImageFile(w, r, cachePath, imageName, platform)
		return
	}

	log.WithFields(log.Fields{
		"image":    imageName,
		"platform": platform,
	}).Info("Downloading image")
	sfKey := imageName + "_" + platform.String()
	result, err, _ := s.downloadGroup.Do(sfKey, func() (interface{}, error) {
		return DownloadImage(imageName, s.cache.Dir(), platform)
	})
	if err != nil {
		log.WithFields(log.Fields{
			"image": imageName,
		}).WithError(err).Error("Failed to download image")
		errorsTotalMetric.Inc()
		writeJSONError(w, fmt.Sprintf("failed to download image: %v", err), http.StatusInternalServerError)
		return
	}
	imagePath := result.(string)

	s.serveImageFile(w, r, imagePath, imageName, platform)
}

// platformsHandler handles the /platforms endpoint
func (s *Server) platformsHandler(w http.ResponseWriter, r *http.Request) {
	imageName, ok := extractImageName(w, r)
	if !ok {
		return
	}

	platforms, err := GetImagePlatforms(imageName)
	if err != nil {
		log.WithField("image", imageName).WithError(err).Error("Failed to get platforms")
		writeJSONError(w, fmt.Sprintf("failed to get platforms: %v", err), http.StatusInternalServerError)
		return
	}
	if platforms == nil {
		platforms = []Platform{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"platforms": platforms})
}

// extractImageName reads and sanitizes the "name" query parameter, writing an
// error response and returning false if it is missing or invalid.
func extractImageName(w http.ResponseWriter, r *http.Request) (string, bool) {
	imageName := r.URL.Query().Get("name")
	if imageName == "" {
		writeJSONError(w, "missing required 'name' query parameter", http.StatusBadRequest)
		return "", false
	}
	imageName, err := sanitizeImageName(imageName)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("invalid image name: %v", err), http.StatusBadRequest)
		return "", false
	}
	return imageName, true
}

// platformFromRequest parses and validates the os/arch/variant query parameters,
// writing an error response and returning false if any value is invalid.
func platformFromRequest(w http.ResponseWriter, r *http.Request) (Platform, bool) {
	platform := DefaultPlatform()
	for _, field := range []struct {
		param string
		dest  *string
	}{
		{"os", &platform.OS},
		{"arch", &platform.Architecture},
		{"variant", &platform.Variant},
	} {
		if val := r.URL.Query().Get(field.param); val != "" {
			if err := validatePlatformParam(field.param, val); err != nil {
				writeJSONError(w, err.Error(), http.StatusBadRequest)
				return Platform{}, false
			}
			*field.dest = val
		}
	}
	return platform, true
}

// serveImageFile streams an image tar file to the response with Range request support
func (s *Server) serveImageFile(w http.ResponseWriter, r *http.Request, imagePath, imageName string, platform Platform) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.WithError(err).Error("Failed to open image file")
		errorsTotalMetric.Inc()
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.WithError(err).Warn("Failed to close image file")
		}
	}(file)

	if err := os.Chtimes(imagePath, time.Now(), time.Now()); err != nil {
		log.WithFields(log.Fields{
			"path": imagePath,
		}).WithError(err).Warn("Failed to update access time")
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.WithError(err).Error("Failed to stat image file")
		errorsTotalMetric.Inc()
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	filename := s.cache.GetCacheFilename(imageName, platform)

	w.Header().Set(contentTypeHeader, "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)

	log.WithFields(log.Fields{
		"image":    imageName,
		"platform": platform,
		"size":     humanizeBytes(fileInfo.Size()),
	}).Info("Served image")
	pullsCountMetric.Inc()
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set(contentTypeHeader, "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.WithError(err).Warn("Failed to write JSON response")
	}
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

// humanizeBytes converts bytes to a human-readable format
func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
		if exp >= len(units)-1 {
			break
		}
	}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
