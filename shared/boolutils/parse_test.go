package boolutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/boolutils"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()

	assert.True(t, boolutils.Parse("true"))
	assert.True(t, boolutils.Parse("yes"))
	assert.False(t, boolutils.Parse("false"))
	assert.False(t, boolutils.Parse("invalid"))
}

func TestParseDefault(t *testing.T) {
	t.Parallel()

	assert.True(t, boolutils.ParseDefault("", true))
	assert.False(t, boolutils.ParseDefault("", false))
	assert.False(t, boolutils.ParseDefault("false", true))
	assert.True(t, boolutils.ParseDefault("true", false))
}
