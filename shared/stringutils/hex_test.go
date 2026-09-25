package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLowerHex(t *testing.T) {
	t.Parallel()

	assert.True(t, IsLowerHex("0123456789abcdef"))
	assert.False(t, IsLowerHex(""))
	assert.False(t, IsLowerHex("ABCDEF"))
	assert.False(t, IsLowerHex("0x1f"))
	assert.False(t, IsLowerHex("abc g"))
}

func TestIsLowerHexOfLength(t *testing.T) {
	t.Parallel()

	assert.True(t, IsLowerHexOfLength("00f067aa0ba902b7", 16))
	assert.False(t, IsLowerHexOfLength("00f067aa0ba902b", 16))
	assert.False(t, IsLowerHexOfLength("00F067AA0BA902B7", 16))
}

func TestIsAllZeros(t *testing.T) {
	t.Parallel()

	assert.True(t, IsAllZeros("0000"))
	assert.False(t, IsAllZeros(""))
	assert.False(t, IsAllZeros("0001"))
}
