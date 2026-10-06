package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluralize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "load", Pluralize("load", "loads", 1))
	assert.Equal(t, "loads", Pluralize("load", "loads", 0))
	assert.Equal(t, "loads", Pluralize("load", "loads", 2))
	assert.Equal(t, "delivers", Pluralize("delivers", "deliver", 1))
	assert.Equal(t, "deliver", Pluralize("delivers", "deliver", 7))
}

func TestCountNoun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1 load", CountNoun(1, "load", "loads"))
	assert.Equal(t, "0 loads", CountNoun(0, "load", "loads"))
	assert.Equal(t, "14 loads", CountNoun(14, "load", "loads"))
	assert.Equal(t, "-1 loads", CountNoun(-1, "load", "loads"))
}
