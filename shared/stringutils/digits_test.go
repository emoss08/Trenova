package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDigitsOnly(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "5551234567", DigitsOnly("(555) 123-4567"))
	assert.Equal(t, "277621", DigitsOnly("MC-277621"))
	assert.Empty(t, DigitsOnly("none"))
}

func TestCollapseUpper(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ACME FREIGHT LLC", CollapseUpper("  acme   freight llc "))
	assert.Empty(t, CollapseUpper("   "))
}
