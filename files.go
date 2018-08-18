package dockerimagesave

import (
	"log"
	"os"
)

func GetFileSize(afile string) int64 {
	fi, err := os.Stat(afile)
	if err != nil {
		log.Print(err)
	}

	return fi.Size()
}

func FileExists(afile string) bool {
	if _, err := os.Stat(afile); os.IsNotExist(err) {
		return false
	}
	return true
}
