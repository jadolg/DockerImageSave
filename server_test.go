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
	"strings"
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

		server.serveImageFile(w, req, testFile, "test:image")

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

		server.serveImageFile(w, req, testFile, "test:image")

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
		server.serveImageFile(w1, req1, testFile, "test:image")
		combined.Write(w1.Body.Bytes())

		req2 := httptest.NewRequest(http.MethodGet, "/image", nil)
		req2.Header.Set("Range", "bytes=10-")
		w2 := httptest.NewRecorder()
		server.serveImageFile(w2, req2, testFile, "test:image")
		combined.Write(w2.Body.Bytes())

		if !bytes.Equal(combined.Bytes(), testContent) {
			t.Errorf("combined content does not match original\nexpected: %s\ngot: %s",
				string(testContent), combined.String())
		}
	})

	t.Run("FullDownload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/image", nil)
		w := httptest.NewRecorder()

		server.serveImageFile(w, req, testFile, "test:image")

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

	server.serveImageFile(w, req, testFile, "test:image")

	resp := w.Result()
	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("expected status 416, got %d", resp.StatusCode)
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes less than 1KB",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "exactly 1KB",
			bytes:    1024,
			expected: "1.00 KB",
		},
		{
			name:     "kilobytes",
			bytes:    5120,
			expected: "5.00 KB",
		},
		{
			name:     "megabytes",
			bytes:    5242880, // 5 MB
			expected: "5.00 MB",
		},
		{
			name:     "megabytes with decimals",
			bytes:    7654321,
			expected: "7.30 MB",
		},
		{
			name:     "gigabytes",
			bytes:    5368709120, // 5 GB
			expected: "5.00 GB",
		},
		{
			name:     "gigabytes with decimals",
			bytes:    1610612736, // 1.5 GB
			expected: "1.50 GB",
		},
		{
			name:     "terabytes",
			bytes:    5497558138880, // 5 TB
			expected: "5.00 TB",
		},
		{
			name:     "petabytes",
			bytes:    5629499534213120, // 5 PB
			expected: "5.00 PB",
		},
		{
			name:     "1023 bytes",
			bytes:    1023,
			expected: "1023 B",
		},
		{
			name:     "1 byte",
			bytes:    1,
			expected: "1 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanizeBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("humanizeBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestHumanizeBytes_FormattingConsistency(t *testing.T) {
	// Test that all results use 2 decimal places (except for bytes)
	testCases := []int64{
		1024,         // 1.00 KB
		1048576,      // 1.00 MB
		1073741824,   // 1.00 GB
		1099511627776, // 1.00 TB
	}

	for _, bytes := range testCases {
		result := humanizeBytes(bytes)
		if !strings.Contains(result, ".") {
			t.Errorf("humanizeBytes(%d) = %s, expected decimal point for formatted size", bytes, result)
		}
	}

	// Test that byte values don't have decimal points
	result := humanizeBytes(512)
	if strings.Contains(result, ".") {
		t.Errorf("humanizeBytes(512) = %s, expected no decimal point for byte values", result)
	}
}

var _ = fmt.Sprintf
