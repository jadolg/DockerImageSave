package dockerimagesave

// https://golangcode.com/create-zip-files-in-go/

import (
	"archive/zip"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// ZipFiles compresses one or many files into a single zip archive file
func ZipFiles(filename string, files []string) error {
	filename = RemoveDoubleDots(filename)
	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func(newfile *os.File) {
		err := newfile.Close()
		if err != nil {
			log.Error(err)
		}
	}(newfile)

	zipWriter := zip.NewWriter(newfile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			log.Error(err)
		}
	}(zipWriter)

	// Add files to zip
	for _, file := range files {
		zipfile, err := os.Open(RemoveDoubleDots(file))
		if err != nil {
			return err
		}
		defer func(zipfile *os.File) {
			err := zipfile.Close()
			if err != nil {
				log.Error(err)
			}
		}(zipfile)

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, zipfile)
		if err != nil {
			return err
		}
	}
	return nil
}
