package dockerimagesave

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizer(t *testing.T) {
	s := "test string\n\r"
	assert.Equal(t, "test string", Sanitize(s))
}
