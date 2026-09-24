package intutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilIfZero(t *testing.T) {
	t.Parallel()

	assert.Nil(t, NilIfZero(0))
	assert.Nil(t, NilIfZero(int64(0)))

	got := NilIfZero(768)
	if assert.NotNil(t, got) {
		assert.Equal(t, 768, *got)
	}
}

func TestMaxPointer(t *testing.T) {
	t.Parallel()

	low, high := int64(10), int64(20)

	assert.Nil(t, MaxPointer[int64](nil, nil))
	assert.Equal(t, int64(10), *MaxPointer(&low, nil))
	assert.Equal(t, int64(20), *MaxPointer(nil, &high))
	assert.Equal(t, int64(20), *MaxPointer(&low, &high))
	assert.Equal(t, int64(20), *MaxPointer(&high, &low))

	got := MaxPointer(&low, nil)
	*got = 99
	assert.Equal(t, int64(10), low, "the result is a copy, never an alias of an input")
}
