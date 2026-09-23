package typeutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualPtr(t *testing.T) {
	t.Parallel()

	one, alsoOne, two := int64(1), int64(1), int64(2)

	assert.True(t, EqualPtr[int64](nil, nil))
	assert.True(t, EqualPtr(&one, &alsoOne), "values are compared, not addresses")
	assert.False(t, EqualPtr(&one, &two))
	assert.False(t, EqualPtr(&one, nil))
	assert.False(t, EqualPtr(nil, &one))
}
