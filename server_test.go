package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	server := NewServer(":8080", "")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if body != "OK\n" {
		t.Errorf("expected body 'OK\\n', got '%s'", body)
	}
}

func TestImageHandler_MissingName(t *testing.T) {
	server := NewServer(":8080", "")

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	w := httptest.NewRecorder()

	server.imageHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestImageHandler_DownloadImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	server := NewServer(":8080", "")

	req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest", nil)
	w := httptest.NewRecorder()

	server.imageHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/gzip" {
		t.Errorf("expected Content-Type 'application/gzip', got '%s'", contentType)
	}

	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("expected Content-Disposition header")
	}

	if w.Body.Len() == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestServeImageFile_RangeRequest(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-range-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	testContent := []byte("0123456789ABCDEFGHIJ")
	testFile := filepath.Join(tempDir, "test_image.tar.gz")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	server := NewServerWithCache(":8080", tempDir)

	t.Run("FirstHalf", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/image", nil)
		req.Header.Set("Range", "bytes=0-9")
		w := httptest.NewRecorder()

		server.serveImageFile(w, req, testFile, "test:image", "")

		resp := w.Result()
		if resp.StatusCode != http.StatusPartialContent {
			t.Errorf("expected status 206, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "0123456789"
		if string(body) != expected {
			t.Errorf("expected body '%s', got '%s'", expected, string(body))
		}

		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "bytes 0-9/20" {
			t.Errorf("expected Content-Range 'bytes 0-9/20', got '%s'", contentRange)
		}
	})

	t.Run("SecondHalf", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/image", nil)
		req.Header.Set("Range", "bytes=10-19")
		w := httptest.NewRecorder()

		server.serveImageFile(w, req, testFile, "test:image", "")

		resp := w.Result()
		if resp.StatusCode != http.StatusPartialContent {
			t.Errorf("expected status 206, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expected := "ABCDEFGHIJ"
		if string(body) != expected {
			t.Errorf("expected body '%s', got '%s'", expected, string(body))
		}

		contentRange := resp.Header.Get("Content-Range")
		if contentRange != "bytes 10-19/20" {
			t.Errorf("expected Content-Range 'bytes 10-19/20', got '%s'", contentRange)
		}
	})

	t.Run("CombineHalves", func(t *testing.T) {
		var combined bytes.Buffer

		req1 := httptest.NewRequest(http.MethodGet, "/image", nil)
		req1.Header.Set("Range", "bytes=0-9")
		w1 := httptest.NewRecorder()
		server.serveImageFile(w1, req1, testFile, "test:image", "")
		combined.Write(w1.Body.Bytes())

		req2 := httptest.NewRequest(http.MethodGet, "/image", nil)
		req2.Header.Set("Range", "bytes=10-")
		w2 := httptest.NewRecorder()
		server.serveImageFile(w2, req2, testFile, "test:image", "")
		combined.Write(w2.Body.Bytes())

		if !bytes.Equal(combined.Bytes(), testContent) {
			t.Errorf("combined content does not match original\nexpected: %s\ngot: %s",
				string(testContent), combined.String())
		}
	})

	t.Run("FullDownload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/image", nil)
		w := httptest.NewRecorder()

		server.serveImageFile(w, req, testFile, "test:image", "")

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !bytes.Equal(body, testContent) {
			t.Errorf("expected full content, got '%s'", string(body))
		}

		acceptRanges := resp.Header.Get("Accept-Ranges")
		if acceptRanges != "bytes" {
			t.Errorf("expected Accept-Ranges 'bytes', got '%s'", acceptRanges)
		}
	})
}

func TestServeImageFile_InvalidRange(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-invalid-range-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	testContent := []byte("0123456789")
	testFile := filepath.Join(tempDir, "test.tar.gz")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	server := NewServerWithCache(":8080", tempDir)

	req := httptest.NewRequest(http.MethodGet, "/image", nil)
	req.Header.Set("Range", "bytes=100-200")
	w := httptest.NewRecorder()

	server.serveImageFile(w, req, testFile, "test:image", "")

	resp := w.Result()
	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("expected status 416, got %d", resp.StatusCode)
	}
}

func TestImageHandler_WithPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	server := NewServer(":8080", "")

	// Test with linux/amd64 platform
	req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest&platform=linux/amd64", nil)
	w := httptest.NewRecorder()

	server.imageHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/gzip" {
		t.Errorf("expected Content-Type 'application/gzip', got '%s'", contentType)
	}
}

func TestImageHandler_WithURLEncodedPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	server := NewServer(":8080", "")

	// Test with URL-encoded platform (linux%2Famd64)
	req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest&platform=linux%2Famd64", nil)
	w := httptest.NewRecorder()

	server.imageHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestImageHandler_InvalidPlatform(t *testing.T) {
	server := NewServer(":8080", "")

	tests := []struct {
		name     string
		platform string
	}{
		{"invalid format", "invalid"},
		{"unsupported OS", "bsd/amd64"},
		{"unsupported arch", "linux/mips"},
		{"empty parts", "/amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest&platform="+tt.platform, nil)
			w := httptest.NewRecorder()

			server.imageHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if body["error"] == "" {
				t.Error("expected error message in response")
			}
		})
	}
}

func TestSanitizePlatform(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		expected  string
		expectErr bool
	}{
		{"linux/amd64", "linux/amd64", "linux/amd64", false},
		{"linux/arm64", "linux/arm64", "linux/arm64", false},
		{"linux/arm", "linux/arm", "linux/arm", false},
		{"linux/386", "linux/386", "linux/386", false},
		{"linux/ppc64le", "linux/ppc64le", "linux/ppc64le", false},
		{"linux/s390x", "linux/s390x", "linux/s390x", false},
		{"linux/riscv64", "linux/riscv64", "linux/riscv64", false},
		{"windows/amd64", "windows/amd64", "windows/amd64", false},
		{"darwin/amd64", "darwin/amd64", "darwin/amd64", false},
		{"darwin/arm64", "darwin/arm64", "darwin/arm64", false},
		{"invalid format", "invalid", "", true},
		{"unsupported OS", "bsd/amd64", "", true},
		{"unsupported arch", "linux/mips", "", true},
		{"too many parts", "linux/amd64/v2", "", true},
		{"empty OS", "/amd64", "", true},
		{"empty arch", "linux/", "", true},
		{"path traversal attempt", "../../../etc/passwd", "", true},
		{"path traversal in os", "../linux/amd64", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizePlatform(tt.platform)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for platform '%s', got nil", tt.platform)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for platform '%s': %v", tt.platform, err)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("expected sanitized platform '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetCacheFilename_WithPlatform(t *testing.T) {
	server := NewServerWithCache(":8080", "")

	tests := []struct {
		name      string
		imageName string
		platform  string
		expected  string
	}{
		{
			name:      "default platform",
			imageName: "alpine:latest",
			platform:  "",
			expected:  "library_alpine_latest_linux_amd64.tar.gz",
		},
		{
			name:      "linux/amd64",
			imageName: "alpine:latest",
			platform:  "linux/amd64",
			expected:  "library_alpine_latest_linux_amd64.tar.gz",
		},
		{
			name:      "linux/arm64",
			imageName: "alpine:latest",
			platform:  "linux/arm64",
			expected:  "library_alpine_latest_linux_arm64.tar.gz",
		},
		{
			name:      "custom registry with platform",
			imageName: "gcr.io/myproject/myimage:v1.0",
			platform:  "linux/arm64",
			expected:  "myproject_myimage_v1.0_linux_arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.getCacheFilename(tt.imageName, tt.platform)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestImageHandler_PlatformNormalization(t *testing.T) {
	// This test verifies that requests without platform and with explicit "linux/amd64"
	// result in the same cache behavior (same filename)
	server := NewServerWithCache(":8080", "")

	// Both should produce the same cache filename
	filenameEmpty := server.getCacheFilename("alpine:latest", "")
	filenameExplicit := server.getCacheFilename("alpine:latest", "linux/amd64")

	if filenameEmpty != filenameExplicit {
		t.Errorf("cache filenames should match for empty and explicit linux/amd64 platform\nempty: %s\nexplicit: %s",
			filenameEmpty, filenameExplicit)
	}

	// Verify the normalized filename format
	expected := "library_alpine_latest_linux_amd64.tar.gz"
	if filenameEmpty != expected {
		t.Errorf("expected filename '%s', got '%s'", expected, filenameEmpty)
	}
}

func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal string", "alpine", "alpine"},
		{"with forward slash", "library/alpine", "library_alpine"},
		{"with backslash", "library\\alpine", "library_alpine"},
		{"path traversal", "../../../etc/passwd", "___etc_passwd"},
		{"double dots", "foo..bar", "foobar"},
		{"leading dot", ".hidden", "hidden"},
		{"multiple leading dots", "...test", "test"},
		{"complex traversal", "../../foo/../bar", "__foo__bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePathComponent(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePathComponent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidatePathContainment(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		fullPath  string
		expectErr bool
	}{
		{"valid path", "/cache", "/cache/file.tar.gz", false},
		{"valid nested path", "/cache", "/cache/subdir/file.tar.gz", false},
		{"path traversal", "/cache", "/cache/../etc/passwd", true},
		{"absolute escape", "/cache", "/etc/passwd", true},
		{"same as base", "/cache", "/cache", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathContainment(tt.basePath, tt.fullPath)
			if tt.expectErr && err == nil {
				t.Errorf("expected error for basePath=%q, fullPath=%q", tt.basePath, tt.fullPath)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error for basePath=%q, fullPath=%q: %v", tt.basePath, tt.fullPath, err)
			}
		})
	}
}

func TestNewServerWithConfig_Auth(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}

	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	server := NewServerWithConfig(":8080", tempDir, authConfig)

	if server.auth == nil {
		t.Fatal("expected auth middleware to be set")
	}

	if !server.auth.IsEnabled() {
		t.Error("expected auth to be enabled")
	}
}

func TestNewServerWithConfig_NoAuth(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	server := NewServerWithConfig(":8080", tempDir, nil)

	if server.auth == nil {
		t.Fatal("expected auth middleware to exist even when nil config")
	}

	if server.auth.IsEnabled() {
		t.Error("expected auth to be disabled when config is nil")
	}
}

func TestImageHandler_WithAuth_Unauthorized(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	authConfig := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}
	server := NewServerWithConfig(":8080", tempDir, authConfig)

	// Wrap imageHandler with auth
	handler := server.auth.WrapFunc(server.imageHandler)

	req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestImageHandler_WithAuth_BasicAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	authConfig := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}
	server := NewServerWithConfig(":8080", tempDir, authConfig)

	handler := server.auth.WrapFunc(server.imageHandler)

	req := httptest.NewRequest(http.MethodGet, "/image?name=alpine:latest", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()

	handler(w, req)

	// With valid auth, should proceed (might get 200 or other status depending on download)
	if w.Code == http.StatusUnauthorized {
		t.Error("expected authenticated request to not return 401")
	}
}

func TestImageHandler_WithAuth_APIKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	authConfig := &AuthConfig{
		Enabled: true,
		APIKeys: []string{"test-api-key-123"},
	}
	server := NewServerWithConfig(":8080", tempDir, authConfig)

	handler := server.auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
	}{
		{
			name: "valid API key in header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "test-api-key-123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid API key in header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "wrong-key")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "no authentication",
			setupRequest:   func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestImageHandler_WithAuth_APIKeyQueryParam(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	authConfig := &AuthConfig{
		Enabled: true,
		APIKeys: []string{"query-key-456"},
	}
	server := NewServerWithConfig(":8080", tempDir, authConfig)

	handler := server.auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Valid API key in query parameter
	req := httptest.NewRequest(http.MethodGet, "/test?api_key=query-key-456", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 with valid API key in query, got %d", w.Code)
	}

	// Invalid API key in query parameter
	req = httptest.NewRequest(http.MethodGet, "/test?api_key=wrong-key", nil)
	w = httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 with invalid API key in query, got %d", w.Code)
	}
}

func TestHealthHandler_NoAuth(t *testing.T) {
	// Health endpoint should be accessible without auth
	tempDir, err := os.MkdirTemp("", "server-auth-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	authConfig := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}
	server := NewServerWithConfig(":8080", tempDir, authConfig)

	// healthHandler is NOT wrapped with auth
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected health endpoint to return 200 without auth, got %d", w.Code)
	}
}

var _ = fmt.Sprintf
