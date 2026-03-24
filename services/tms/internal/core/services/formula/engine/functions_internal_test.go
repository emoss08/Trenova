package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToFloat64_AllBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    float64
		wantErr bool
	}{
		{name: "float64", input: float64(3.14), want: 3.14},
		{name: "float32", input: float32(2.5), want: 2.5},
		{name: "int", input: int(42), want: 42.0},
		{name: "int64", input: int64(100), want: 100.0},
		{name: "int32", input: int32(50), want: 50.0},
		{name: "string", input: "abc", wantErr: true},
		{name: "bool", input: true, wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "slice", input: []int{1}, wantErr: true},
		{name: "map", input: map[string]int{"a": 1}, wantErr: true},
		{name: "uint", input: uint(5), wantErr: true},
		{name: "uint64", input: uint64(10), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := toFloat64(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot convert")
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestToInt_AllBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		want    int
		wantErr bool
	}{
		{name: "int", input: int(42), want: 42},
		{name: "int64", input: int64(100), want: 100},
		{name: "int32", input: int32(50), want: 50},
		{name: "float64", input: float64(3.9), want: 3},
		{name: "string", input: "abc", wantErr: true},
		{name: "bool", input: true, wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "float32", input: float32(2.5), wantErr: true},
		{name: "uint", input: uint(5), wantErr: true},
		{name: "slice", input: []int{1, 2}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := toInt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot convert")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCeilFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid float64", func(t *testing.T) {
		t.Parallel()
		got, err := ceilFn(3.2)
		require.NoError(t, err)
		assert.Equal(t, 4.0, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		_, err := ceilFn("abc")
		require.Error(t, err)
	})
}

func TestFloorFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid float64", func(t *testing.T) {
		t.Parallel()
		got, err := floorFn(3.7)
		require.NoError(t, err)
		assert.Equal(t, 3.0, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		_, err := floorFn("abc")
		require.Error(t, err)
	})
}

func TestAbsFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid float64", func(t *testing.T) {
		t.Parallel()
		got, err := absFn(-5.0)
		require.NoError(t, err)
		assert.Equal(t, 5.0, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		_, err := absFn(true)
		require.Error(t, err)
	})
}

func TestSqrtFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid float64", func(t *testing.T) {
		t.Parallel()
		got, err := sqrtFn(9.0)
		require.NoError(t, err)
		assert.Equal(t, 3.0, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		_, err := sqrtFn("nine")
		require.Error(t, err)
	})
}

func TestPowFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		got, err := powFn(2.0, 3.0)
		require.NoError(t, err)
		assert.Equal(t, 8.0, got)
	})

	t.Run("invalid base", func(t *testing.T) {
		t.Parallel()
		_, err := powFn("two", 3.0)
		require.Error(t, err)
	})

	t.Run("invalid exponent", func(t *testing.T) {
		t.Parallel()
		_, err := powFn(2.0, "three")
		require.Error(t, err)
	})
}

func TestMinFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid a less", func(t *testing.T) {
		t.Parallel()
		got, err := minFn(2.0, 5.0)
		require.NoError(t, err)
		assert.Equal(t, 2.0, got)
	})

	t.Run("valid b less", func(t *testing.T) {
		t.Parallel()
		got, err := minFn(5.0, 2.0)
		require.NoError(t, err)
		assert.Equal(t, 2.0, got)
	})

	t.Run("invalid first arg", func(t *testing.T) {
		t.Parallel()
		_, err := minFn("abc", 5.0)
		require.Error(t, err)
	})

	t.Run("invalid second arg", func(t *testing.T) {
		t.Parallel()
		_, err := minFn(5.0, "xyz")
		require.Error(t, err)
	})
}

func TestMaxFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid a greater", func(t *testing.T) {
		t.Parallel()
		got, err := maxFn(5.0, 2.0)
		require.NoError(t, err)
		assert.Equal(t, 5.0, got)
	})

	t.Run("valid b greater", func(t *testing.T) {
		t.Parallel()
		got, err := maxFn(2.0, 5.0)
		require.NoError(t, err)
		assert.Equal(t, 5.0, got)
	})

	t.Run("invalid first arg", func(t *testing.T) {
		t.Parallel()
		_, err := maxFn("abc", 5.0)
		require.Error(t, err)
	})

	t.Run("invalid second arg", func(t *testing.T) {
		t.Parallel()
		_, err := maxFn(5.0, true)
		require.Error(t, err)
	})
}

func TestClampFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("value in range", func(t *testing.T) {
		t.Parallel()
		got, err := clampFn(5.0, 0.0, 10.0)
		require.NoError(t, err)
		assert.Equal(t, 5.0, got)
	})

	t.Run("value below min", func(t *testing.T) {
		t.Parallel()
		got, err := clampFn(-5.0, 0.0, 10.0)
		require.NoError(t, err)
		assert.Equal(t, 0.0, got)
	})

	t.Run("value above max", func(t *testing.T) {
		t.Parallel()
		got, err := clampFn(15.0, 0.0, 10.0)
		require.NoError(t, err)
		assert.Equal(t, 10.0, got)
	})

	t.Run("invalid value", func(t *testing.T) {
		t.Parallel()
		_, err := clampFn("five", 0.0, 10.0)
		require.Error(t, err)
	})

	t.Run("invalid min", func(t *testing.T) {
		t.Parallel()
		_, err := clampFn(5.0, "zero", 10.0)
		require.Error(t, err)
	})

	t.Run("invalid max", func(t *testing.T) {
		t.Parallel()
		_, err := clampFn(5.0, 0.0, "ten")
		require.Error(t, err)
	})
}

func TestSumFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		got, err := sumFn(1.0, 2.0, 3.0)
		require.NoError(t, err)
		assert.Equal(t, 6.0, got)
	})

	t.Run("invalid value", func(t *testing.T) {
		t.Parallel()
		_, err := sumFn(1.0, "two", 3.0)
		require.Error(t, err)
	})
}

func TestAvgFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("no args returns zero", func(t *testing.T) {
		t.Parallel()
		got, err := avgFn()
		require.NoError(t, err)
		assert.Equal(t, 0.0, got)
	})

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		got, err := avgFn(2.0, 4.0, 6.0)
		require.NoError(t, err)
		assert.Equal(t, 4.0, got)
	})

	t.Run("invalid value propagates error", func(t *testing.T) {
		t.Parallel()
		_, err := avgFn(1.0, "bad", 3.0)
		require.Error(t, err)
	})
}

func TestRoundFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("valid no decimals", func(t *testing.T) {
		t.Parallel()
		got, err := roundFn(3.7)
		require.NoError(t, err)
		assert.InDelta(t, 4.0, got, 0.001)
	})

	t.Run("valid with decimals", func(t *testing.T) {
		t.Parallel()
		got, err := roundFn(3.14159, 2)
		require.NoError(t, err)
		assert.InDelta(t, 3.14, got, 0.001)
	})

	t.Run("invalid value", func(t *testing.T) {
		t.Parallel()
		_, err := roundFn("abc")
		require.Error(t, err)
	})

	t.Run("invalid decimals", func(t *testing.T) {
		t.Parallel()
		_, err := roundFn(3.14, "two")
		require.Error(t, err)
	})
}

func TestCoalesceFn_Direct(t *testing.T) {
	t.Parallel()

	t.Run("first non-nil", func(t *testing.T) {
		t.Parallel()
		got, err := coalesceFn(nil, nil, 42.0)
		require.NoError(t, err)
		assert.Equal(t, 42.0, got)
	})

	t.Run("all nil returns error", func(t *testing.T) {
		t.Parallel()
		_, err := coalesceFn(nil, nil, nil)
		require.Error(t, err)
	})

	t.Run("first is valid", func(t *testing.T) {
		t.Parallel()
		got, err := coalesceFn("hello", nil, 42.0)
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})
}
