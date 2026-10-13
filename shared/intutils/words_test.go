package intutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSmallNumberWord(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "zero", SmallNumberWord(0))
	assert.Equal(t, "two", SmallNumberWord(2))
	assert.Equal(t, "twelve", SmallNumberWord(12))
	assert.Equal(t, "13", SmallNumberWord(13))
	assert.Equal(t, "-1", SmallNumberWord(-1))
}
