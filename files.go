package main

import (
	"log"
	"os"
)

func getFileSize(afile string) int64 {
	fi, err := os.Stat(afile)
	if err != nil {
		log.Print(err)
	}

	return fi.Size()
}

func fileExists(afile string) bool {
	if _, err := os.Stat(afile); os.IsNotExist(err) {
		return false
	}
	return true
}
