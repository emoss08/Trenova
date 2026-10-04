package passwordutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCommon(t *testing.T) {
	t.Parallel()

	assert.True(t, IsCommon("password1234"))
	assert.True(t, IsCommon("  PASSWORD1234 "))
	assert.True(t, IsCommon("qwertyuiopasdf"))
	assert.True(t, IsCommon("zzzzzzzzzzzz"))
	assert.True(t, IsCommon("abcdefghijklmnop"))
	assert.True(t, IsCommon("987654321098"[:9]))
	assert.False(t, IsCommon("correct-horse-stapler-92"))
	assert.False(t, IsCommon(""))
	assert.False(t, IsCommon("Tr4ck-the-l0ads!"))
}

func TestContainsIgnoringCase(t *testing.T) {
	t.Parallel()

	assert.True(t, ContainsIgnoringCase("MyDana.Whitfield2026", "dana.whitfield"))
	assert.False(t, ContainsIgnoringCase("anything-goes-here", "ab"))
	assert.False(t, ContainsIgnoringCase("anything-goes-here", "dana"))
}
