package stringutils

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateBytes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", TruncateBytes("abc", 0))
	assert.Equal(t, "abc", TruncateBytes("abc", 3))
	assert.Equal(t, "ab", TruncateBytes("abc", 2))

	cut := TruncateBytes("a€b", 3)
	assert.Equal(t, "a", cut, "a rune split by the cap is dropped whole")
	assert.True(t, utf8.ValidString(cut))
	assert.Equal(t, "a€", TruncateBytes("a€b", 4))
}
