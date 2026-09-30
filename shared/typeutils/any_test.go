package typeutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/stretchr/testify/assert"
)

func TestStringOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "text", typeutils.StringOf("text"))
	assert.Equal(t, "  padded  ", typeutils.StringOf("  padded  "))
	assert.Empty(t, typeutils.StringOf(nil))
	assert.Empty(t, typeutils.StringOf(42), "a number out of a JSON document is not a string")
	assert.Empty(t, typeutils.StringOf(true))
}

func TestStringOfTrimmed(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "padded", typeutils.StringOfTrimmed("  padded  "))
	assert.Empty(t, typeutils.StringOfTrimmed("   "))
	assert.Empty(t, typeutils.StringOfTrimmed(42))
}

func TestBoolOf(t *testing.T) {
	t.Parallel()

	assert.True(t, typeutils.BoolOf(true))
	assert.False(t, typeutils.BoolOf(false))
	assert.False(t, typeutils.BoolOf(nil))
	assert.False(t, typeutils.BoolOf("true"), "a string that reads true is still not a bool")
}
