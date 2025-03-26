package dockerimagesave

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFileSize(t *testing.T) {
	assert.Equal(t, int64(1392), GetFileSize("zipfile.go"))
}

func TestFileExists(t *testing.T) {
	assert.True(t, FileExists("zipfile.go"))
}
