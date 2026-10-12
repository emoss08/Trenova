package intutils_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/stretchr/testify/assert"
)

func TestMax(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 9, intutils.Max(4, 9))
	assert.Equal(t, 9, intutils.Max(9, 4))
	assert.Equal(t, 7, intutils.Max(7, 7))
	assert.Equal(t, int64(-2), intutils.Max(int64(-2), int64(-8)))
}

func TestMin(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4, intutils.Min(4, 9))
	assert.Equal(t, 4, intutils.Min(9, 4))
	assert.Equal(t, 7, intutils.Min(7, 7))
	assert.Equal(t, int64(-8), intutils.Min(int64(-2), int64(-8)))
}

func TestAbs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(5), intutils.Abs(int64(5)))
	assert.Equal(t, int64(5), intutils.Abs(int64(-5)))
	assert.Equal(t, int64(0), intutils.Abs(int64(0)))
	assert.Equal(
		t,
		int64(math.MaxInt64),
		intutils.Abs(int64(math.MinInt64)+1),
		"a large negative keeps its magnitude",
	)
	assert.Equal(
		t,
		int64(math.MinInt64),
		intutils.Abs(int64(math.MinInt64)),
		"the smallest int64 has no positive counterpart and is returned unchanged",
	)
}

func TestFirstPositive(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, intutils.FirstPositive[int]())
	assert.Equal(t, 0, intutils.FirstPositive(0, -3))
	assert.Equal(t, 8192, intutils.FirstPositive(0, -1, 8192, 4096))
}
