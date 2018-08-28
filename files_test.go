package dockerimagesave

import (
	"testing"
)

func TestGetFileSize(t *testing.T) {
	if GetFileSize("zipfile.go") != 1033 {
		t.Fail()
	}
}

func TestFileExists(t *testing.T) {
	if !FileExists("zipfile.go") {
		t.Fail()
	}
}
