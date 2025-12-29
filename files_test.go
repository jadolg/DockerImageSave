package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDecompressGzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-decompress-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	content := []byte("Hello, this is test content for gzip decompression!")
	gzPath := filepath.Join(tempDir, "test.gz")
	outPath := filepath.Join(tempDir, "test.out")

	gzFile, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}

	gzWriter := gzip.NewWriter(gzFile)
	if _, err := gzWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	gzWriter.Close()
	gzFile.Close()

	if err := decompressGzip(gzPath, outPath); err != nil {
		t.Fatalf("decompressGzip failed: %v", err)
	}

	result, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result, content) {
		t.Errorf("content mismatch: got %q, want %q", result, content)
	}
}

func TestDecompressGzip_NonGzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-decompress-nongz-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	content := []byte("This is plain text, not gzipped")
	srcPath := filepath.Join(tempDir, "plain.txt")
	outPath := filepath.Join(tempDir, "plain.out")

	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := decompressGzip(srcPath, outPath); err != nil {
		t.Fatalf("decompressGzip failed for non-gzip: %v", err)
	}

	result, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result, content) {
		t.Errorf("content mismatch: got %q, want %q", result, content)
	}
}

func TestCreateTar(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "test-tar-src-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, srcDir)

	file1Content := []byte("file1 content")
	file2Content := []byte("file2 content")

	if err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), file1Content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "file2.txt"), file2Content, 0644); err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(os.TempDir(), "test-output.tar")
	defer os.Remove(tarPath)

	if err := createTar(srcDir, tarPath); err != nil {
		t.Fatalf("createTar failed: %v", err)
	}

	tarFile, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tarFile.Close()

	tr := tar.NewReader(tarFile)
	files := make(map[string][]byte)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if !hdr.FileInfo().IsDir() {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			files[hdr.Name] = content
		}
	}

	if !bytes.Equal(files["file1.txt"], file1Content) {
		t.Errorf("file1.txt content mismatch")
	}
	if !bytes.Equal(files["file2.txt"], file2Content) {
		t.Errorf("file2.txt content mismatch")
	}
}

func TestCreateTar_NestedDirectories(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "test-tar-nested-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, srcDir)

	subDir := filepath.Join(srcDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}

	tarPath := filepath.Join(os.TempDir(), "test-nested.tar")
	defer os.Remove(tarPath)

	if err := createTar(srcDir, tarPath); err != nil {
		t.Fatalf("createTar failed: %v", err)
	}

	tarFile, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tarFile.Close()

	tr := tar.NewReader(tarFile)
	foundNested := false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}

		if hdr.Name == filepath.Join("subdir", "nested.txt") {
			foundNested = true
			break
		}
	}

	if !foundNested {
		t.Error("nested file not found in tar archive")
	}
}
