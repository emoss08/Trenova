package intutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoundSignificant(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value  int64
		digits int
		want   int64
	}{
		{0, 2, 0},
		{7, 2, 7},
		{42, 2, 42},
		{99, 2, 99},
		{104, 2, 100},
		{105, 2, 110},
		{1234, 2, 1200},
		{1250, 2, 1300},
		{98765, 2, 99000},
		{-1250, 2, -1300},
		{1234, 0, 1234},
		{1234, 1, 1000},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, RoundSignificant(tc.value, tc.digits), "%d at %d", tc.value, tc.digits)
	}
}

func TestRoundedShare(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 35, RoundedShare(33, 100, 5))
	assert.Equal(t, 30, RoundedShare(32, 100, 5))
	assert.Equal(t, 33, RoundedShare(1, 3, 1))
	assert.Equal(t, 67, RoundedShare(2, 3, 1))
	assert.Equal(t, 100, RoundedShare(5, 5, 5))
	assert.Equal(t, 100, RoundedShare(6, 5, 5))
	assert.Equal(t, 0, RoundedShare(0, 5, 5))
	assert.Equal(t, 0, RoundedShare(3, 0, 5))
	assert.Equal(t, 3, RoundedShare(1, 40, 0), "a step of zero is taken as one")
}

func TestRoundToStep(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(65), RoundToStep(63, 5))
	assert.Equal(t, int64(60), RoundToStep(62, 5))
	assert.Equal(t, int64(65), RoundToStep(64, 5))
	assert.Equal(t, int64(-65), RoundToStep(-63, 5))
	assert.Equal(t, int64(63), RoundToStep(63, 1))
	assert.Equal(t, int64(63), RoundToStep(63, 0))
}
