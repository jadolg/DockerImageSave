package dockerimagesave

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPullImage(t *testing.T) {
	err := PullImage("busybox:1.29.2")
	assert.NoError(t, err)
}

func TestSaveImage(t *testing.T) {
	err := SaveImage("busybox:1.29.2", "/tmp")
	assert.NoError(t, err)
	assert.FileExists(t, "/tmp/busybox_1.29.2.tar")
}

func TestImageExists(t *testing.T) {
	exists, err := ImageExists("busybox:1.29.2")
	assert.True(t, exists)
	assert.NoError(t, err)
	exists, err = ImageExists("nothing_here:latest")
	assert.False(t, exists)
	assert.NoError(t, err)
}

func TestImageExistsInRegistry(t *testing.T) {
	exists, err := ImageExistsInRegistry("busybox:1.29.2")
	assert.True(t, exists)
	assert.NoError(t, err)

	exists, err = ImageExistsInRegistry("qweqwe:1")
	assert.False(t, exists)
	assert.NoError(t, err)
}
