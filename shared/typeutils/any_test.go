package typeutils_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/stretchr/testify/assert"
)

func TestStringOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "text", typeutils.StringOf("text"))
	assert.Equal(t, "  padded  ", typeutils.StringOf("  padded  "))
	assert.Empty(t, typeutils.StringOf(nil))
	assert.Empty(t, typeutils.StringOf(42), "a number out of a JSON document is not a string")
	assert.Empty(t, typeutils.StringOf(true))
}

func TestStringOfTrimmed(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "padded", typeutils.StringOfTrimmed("  padded  "))
	assert.Empty(t, typeutils.StringOfTrimmed("   "))
	assert.Empty(t, typeutils.StringOfTrimmed(42))
}

func TestBoolOf(t *testing.T) {
	t.Parallel()

	assert.True(t, typeutils.BoolOf(true))
	assert.False(t, typeutils.BoolOf(false))
	assert.False(t, typeutils.BoolOf(nil))
	assert.False(t, typeutils.BoolOf("true"), "a string that reads true is still not a bool")
}

func TestIntOf(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value any
		want  int
		ok    bool
	}{
		{name: "a whole float from a generic decoder", value: float64(128), want: 128, ok: true},
		{name: "an int", value: 7, want: 7, ok: true},
		{name: "an int64", value: int64(9), want: 9, ok: true},
		{name: "an int32", value: int32(3), want: 3, ok: true},
		{name: "zero", value: float64(0), want: 0, ok: true},
		{name: "a fraction", value: 2.5, ok: false},
		{name: "infinity", value: math.Inf(1), ok: false},
		{name: "a numeric string", value: "12", ok: false},
		{name: "nil", value: nil, ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := typeutils.IntOf(tc.value)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
