package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
)

// closeWithLog closes an io.Closer and logs any error with the given context
func closeWithLog(c io.Closer, context string) {
	if err := c.Close(); err != nil {
		log.Printf("Error closing %s: %v\n", context, err)
	}
}

// decompressGzip decompresses a gzip file to a destination path
func decompressGzip(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer closeWithLog(srcFile, "source file")

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		_, err := srcFile.Seek(0, 0)
		if err != nil {
			return err
		}
		dstFile, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer closeWithLog(dstFile, "destination file")
		_, err = io.Copy(dstFile, srcFile)
		return err
	}
	defer closeWithLog(gzReader, "gzip reader")

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer closeWithLog(dstFile, "destination file")

	hasher := sha256.New()
	writer := io.MultiWriter(dstFile, hasher)
	_, err = io.Copy(writer, gzReader)
	if err != nil {
		return err
	}

	_ = hex.EncodeToString(hasher.Sum(nil))
	return nil
}

// createTar creates a gzip-compressed tar archive from a source directory
func createTar(srcDir, destPath string) error {
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer closeWithLog(file, "tar.gz file")

	gzWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer closeWithLog(gzWriter, "gzip writer")

	tw := tar.NewWriter(gzWriter)
	defer closeWithLog(tw, "tar writer")

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		return copyFileToTar(tw, path)
	})
}

// copyFileToTar copies a single file to a tar writer, ensuring the file is closed immediately after copying
func copyFileToTar(tw *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer closeWithLog(f, "file")

	_, err = io.Copy(tw, f)
	return err
}
