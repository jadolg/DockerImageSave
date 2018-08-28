package dockerimagesave

import (
	"log"
	"os"
)

// GetFileSize gets the size of a file
func GetFileSize(afile string) int64 {
	fi, err := os.Stat(afile)
	if err != nil {
		log.Print(err)
	}

	return fi.Size()
}

//FileExists checks if a file exists
func FileExists(afile string) bool {
	if _, err := os.Stat(afile); os.IsNotExist(err) {
		return false
	}
	return true
}
