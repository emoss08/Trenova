package optional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValue_ApplyWritesOnlyWhenSet(t *testing.T) {
	t.Parallel()

	field := 3
	assert.False(t, Value[int]{}.Apply(&field))
	assert.Equal(t, 3, field)

	assert.True(t, Some(0).Apply(&field))
	assert.Equal(t, 0, field)
}

func TestSome_MarksTheValueSet(t *testing.T) {
	t.Parallel()

	var empty *string
	got := Some(empty)

	assert.True(t, got.Set)
	assert.Nil(t, got.Value)
}
