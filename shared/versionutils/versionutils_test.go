package versionutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	v, err := Parse("v1.12.3")
	require.NoError(t, err)
	assert.Equal(t, Version{Major: 1, Minor: 12, Patch: 3}, v)
	assert.Equal(t, "1.12.3", v.String())

	for _, raw := range []string{"", "1", "1.2", "1.2.3.4", "1.02.3", "1.a.3", "1.-2.3", "1.2.3-beta"} {
		_, err = Parse(raw)
		assert.ErrorIs(t, err, ErrInvalidVersion, raw)
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	a, _ := Parse("1.10.0")
	b, _ := Parse("1.9.9")
	assert.Positive(t, Compare(a, b))
	assert.Negative(t, Compare(b, a))
	assert.Zero(t, Compare(a, a))
}

func TestAtLeast(t *testing.T) {
	t.Parallel()

	assert.True(t, AtLeast("1.2.3", ""))
	assert.True(t, AtLeast("1.2.3", "1.2.3"))
	assert.True(t, AtLeast("2.0.0", "1.9.9"))
	assert.False(t, AtLeast("1.2.2", "1.2.3"))
	assert.False(t, AtLeast("garbage", "1.0.0"))
	assert.True(t, AtLeast("1.0.0", "garbage"))
}
