package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestWithDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fallback", stringutils.WithDefault("", "fallback"))
	assert.Equal(t, "fallback", stringutils.WithDefault("   ", "fallback"))
	assert.Equal(t, "value", stringutils.WithDefault(" value ", "fallback"))
}

func TestSplitCSV(t *testing.T) {
	t.Parallel()

	assert.Empty(t, stringutils.SplitCSV(""))
	assert.Equal(t, []string{"a", "b", "c"}, stringutils.SplitCSV("a, b,, c "))
}
