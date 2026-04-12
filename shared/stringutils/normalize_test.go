package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIdentifier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ABC123", NormalizeIdentifier("Abc-123"))
	assert.Equal(t, "INV2024001", NormalizeIdentifier("INV 2024/001"))
	assert.Equal(t, "", NormalizeIdentifier(" -_/ "))
}
