package rategeo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeTypeValues_AreEveryValidScope(t *testing.T) {
	t.Parallel()

	values := ScopeTypeValues()
	assert.Len(t, values, 9)
	for _, value := range values {
		assert.True(t, value.IsValid(), value)
	}
	assert.False(t, ScopeType("County").IsValid())
}
