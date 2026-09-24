package memtable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionalNumber(t *testing.T) {
	t.Parallel()

	got, ok := OptionalNumber[int64](nil)
	assert.False(t, ok)
	assert.Zero(t, got)

	stamp := int64(1_790_000_000)
	got, ok = OptionalNumber(&stamp)
	assert.True(t, ok)
	assert.InDelta(t, 1_790_000_000, got, 0)

	zero := 0
	got, ok = OptionalNumber(&zero)
	assert.True(t, ok, "a zero value is present, not missing")
	assert.Zero(t, got)
}
