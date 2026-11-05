package securityconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPlaceholder(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"change-me", " CHANGE-ME ", "my-secret", "example-key"} {
		assert.True(t, IsPlaceholder(value), value)
	}
	for _, value := range []string{"", "k9Fq2vXw7LmP0zTb"} {
		assert.False(t, IsPlaceholder(value), value)
	}
}
