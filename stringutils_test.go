package dockerimagesave

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSanitizer(t *testing.T) {
	s := "test string\n\r"
	assert.Equal(t, "test string", Sanitize(s))
}

func TestRemoveDots(t *testing.T) {
	assert.Equal(t, "asd/././ppp.a", RemoveDoubleDots("asd/../../ppp.a"))
	assert.Equal(t, "asd/././ppp.a", RemoveDoubleDots("asd/.../.../ppp.a"))
	assert.Equal(t, "asdppp.a", RemoveDoubleDots("asdppp.a"))
}
