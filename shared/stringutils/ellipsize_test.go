package stringutils

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestEllipsize_NeverExceedsTheLimit(t *testing.T) {
	t.Parallel()

	got := Ellipsize(strings.Repeat("a", 300), 200)
	assert.Equal(t, 200, utf8.RuneCountInString(got))
	assert.True(t, strings.HasSuffix(got, "…"))

	assert.Equal(t, "short", Ellipsize("  short  ", 200))
	assert.Equal(t, "exact", Ellipsize("exact", 5))
	assert.Equal(t, "abc…", Ellipsize("abc defgh", 5))
	assert.Equal(t, "…", Ellipsize("abc", 1))
	assert.Empty(t, Ellipsize("abc", 0))
}
