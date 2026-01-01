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

var _ = fmt.Sprintf
