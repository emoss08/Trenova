package intutils_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeIntToUint32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected uint32
	}{
		{"zero", 0, 0},
		{"positive value", 100, 100},
		{"max uint32", math.MaxUint32, math.MaxUint32},
		{"negative value returns zero", -1, 0},
		{"large negative returns zero", -1000000, 0},
		{"exceeds max uint32 returns zero", math.MaxUint32 + 1, 0},
		{"max int returns zero on 64-bit", math.MaxInt, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := intutils.SafeIntToUint32(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeUint64ToInt64(t *testing.T) {
	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()
		result, err := intutils.SafeUint64ToInt64(0)
		require.NoError(t, err)
		assert.Equal(t, int64(0), result)
	})

	t.Run("positive value within range", func(t *testing.T) {
		t.Parallel()
		result, err := intutils.SafeUint64ToInt64(100)
		require.NoError(t, err)
		assert.Equal(t, int64(100), result)
	})

	t.Run("max int64 value", func(t *testing.T) {
		t.Parallel()
		result, err := intutils.SafeUint64ToInt64(math.MaxInt64)
		require.NoError(t, err)
		assert.Equal(t, int64(math.MaxInt64), result)
	})

	t.Run("exceeds max int64 returns error", func(t *testing.T) {
		t.Parallel()
		result, err := intutils.SafeUint64ToInt64(math.MaxInt64 + 1)
		require.Error(t, err)
		assert.Equal(t, int64(0), result)
		assert.Contains(t, err.Error(), "outside int64 range")
	})

	t.Run("max uint64 returns error", func(t *testing.T) {
		t.Parallel()
		result, err := intutils.SafeUint64ToInt64(math.MaxUint64)
		require.Error(t, err)
		assert.Equal(t, int64(0), result)
	})
}

func TestWithDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      int
		def      int
		expected int
	}{
		{"returns val when non-zero positive", 42, 100, 42},
		{"returns val when non-zero negative", -5, 100, -5},
		{"returns def when val is zero", 0, 100, 100},
		{"returns zero def when val is zero", 0, 0, 0},
		{"returns val when def is zero", 42, 0, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := intutils.WithDefault(tt.val, tt.def)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClamp(t *testing.T) {
	t.Parallel()

	t.Run("int type", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			value    int
			min      int
			max      int
			expected int
		}{
			{"value within range", 5, 0, 10, 5},
			{"value equals min", 0, 0, 10, 0},
			{"value equals max", 10, 0, 10, 10},
			{"value below min", -5, 0, 10, 0},
			{"value above max", 15, 0, 10, 10},
			{"min equals max value below", 3, 5, 5, 5},
			{"min equals max value above", 7, 5, 5, 5},
			{"min equals max value equal", 5, 5, 5, 5},
			{"negative range", -5, -10, -1, -5},
			{"clamp to negative min", -15, -10, -1, -10},
			{"clamp to negative max", 0, -10, -1, -1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result := intutils.Clamp(tt.value, tt.min, tt.max)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("int64 type", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			value    int64
			min      int64
			max      int64
			expected int64
		}{
			{"value within range", 5, 0, 10, 5},
			{"value below min", -5, 0, 10, 0},
			{"value above max", 15, 0, 10, 10},
			{"large values within range", math.MaxInt64 - 1, 0, math.MaxInt64, math.MaxInt64 - 1},
			{"clamp to large max", math.MaxInt64, 0, math.MaxInt64 - 1, math.MaxInt64 - 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				result := intutils.Clamp(tt.value, tt.min, tt.max)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestAbsDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        int64
		b        int64
		expected int64
	}{
		{"same values", 10, 10, 0},
		{"first greater", 15, 10, 5},
		{"second greater", 10, 15, 5},
		{"negative values", -10, -3, 7},
		{"mixed signs", -10, 5, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, intutils.AbsDiff(tt.a, tt.b))
		})
	}
}

func TestSafeShiftAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		n        int
		maxShift int
		expected uint
	}{
		{"zero", 0, 5, 0},
		{"positive within max", 3, 5, 3},
		{"equals max", 5, 5, 5},
		{"exceeds max", 10, 5, 5},
		{"negative returns zero", -1, 5, 0},
		{"large negative returns zero", -100, 5, 0},
		{"large max shift", 10, 63, 10},
		{"exceeds large max", 100, 63, 63},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := intutils.SafeShiftAmount(tt.n, tt.maxShift)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClonePointer(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, intutils.ClonePointer[int64](nil))
	})

	t.Run("int64 value", func(t *testing.T) {
		t.Parallel()
		value := int64(42)
		cloned := intutils.ClonePointer(&value)
		require.NotNil(t, cloned)
		assert.Equal(t, int64(42), *cloned)
		assert.NotSame(t, &value, cloned)
	})

	t.Run("int16 value", func(t *testing.T) {
		t.Parallel()
		value := int16(7)
		cloned := intutils.ClonePointer(&value)
		require.NotNil(t, cloned)
		assert.Equal(t, int16(7), *cloned)
		assert.NotSame(t, &value, cloned)
	})

	t.Run("float64 value", func(t *testing.T) {
		t.Parallel()
		value := 12.5
		cloned := intutils.ClonePointer(&value)
		require.NotNil(t, cloned)
		assert.Equal(t, 12.5, *cloned)
		assert.NotSame(t, &value, cloned)
	})
}
