package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "second", stringutils.FirstNonEmpty("", "  ", "second", "third"))
	assert.Equal(t, "  padded  ", stringutils.FirstNonEmpty("", "  padded  "))
	assert.Empty(t, stringutils.FirstNonEmpty("", "   "))
	assert.Empty(t, stringutils.FirstNonEmpty())
}

func TestFirstNonEmptyTrimmed(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "second", stringutils.FirstNonEmptyTrimmed("", "  ", "second", "third"))
	assert.Equal(t, "padded", stringutils.FirstNonEmptyTrimmed("", "  padded  "))
	assert.Empty(t, stringutils.FirstNonEmptyTrimmed("", "   "))
	assert.Empty(t, stringutils.FirstNonEmptyTrimmed())
}
