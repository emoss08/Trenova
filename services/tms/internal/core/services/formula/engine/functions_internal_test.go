package engine

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
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

	t.Run("negative value", func(t *testing.T) {
		t.Parallel()
		_, err := sqrtFn(-4.0)
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

func TestRoundFn_DecimalsRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		decimals int
		wantErr  bool
	}{
		{name: "upper bound", decimals: 12, wantErr: false},
		{name: "lower bound", decimals: -12, wantErr: false},
		{name: "above upper bound", decimals: 13, wantErr: true},
		{name: "below lower bound", decimals: -13, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := roundFn(3.14159, tt.decimals)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "round decimals must be between")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestToDecimal_AllBranches(t *testing.T) {
	t.Parallel()

	e := &Engine{}

	tests := []struct {
		name      string
		input     any
		want      decimal.Decimal
		wantErr   bool
		wantErrIs error
	}{
		{name: "float64", input: float64(3.14), want: decimal.NewFromFloat(3.14)},
		{name: "float32", input: float32(2.5), want: decimal.NewFromFloat(2.5)},
		{name: "int", input: int(42), want: decimal.NewFromInt(42)},
		{name: "int64", input: int64(100), want: decimal.NewFromInt(100)},
		{name: "int32", input: int32(50), want: decimal.NewFromInt(50)},
		{name: "int16", input: int16(25), want: decimal.NewFromInt(25)},
		{name: "int8", input: int8(12), want: decimal.NewFromInt(12)},
		{name: "uint", input: uint(5), want: decimal.NewFromInt(5)},
		{name: "uint64", input: uint64(10), want: decimal.NewFromInt(10)},
		{name: "uint32", input: uint32(20), want: decimal.NewFromInt(20)},
		{name: "uint16", input: uint16(30), want: decimal.NewFromInt(30)},
		{name: "uint8", input: uint8(40), want: decimal.NewFromInt(40)},
		{name: "decimal", input: decimal.NewFromFloat(9.9), want: decimal.NewFromFloat(9.9)},
		{
			name:  "valid null decimal",
			input: decimal.NullDecimal{Decimal: decimal.NewFromFloat(7.5), Valid: true},
			want:  decimal.NewFromFloat(7.5),
		},
		{name: "bool true", input: true, want: decimal.NewFromInt(1)},
		{name: "bool false", input: false, want: decimal.NewFromInt(0)},
		{
			name:      "invalid null decimal",
			input:     decimal.NullDecimal{},
			wantErr:   true,
			wantErrIs: ErrNullResult,
		},
		{name: "NaN", input: math.NaN(), wantErr: true, wantErrIs: ErrNonFiniteResult},
		{
			name:      "positive infinity",
			input:     math.Inf(1),
			wantErr:   true,
			wantErrIs: ErrNonFiniteResult,
		},
		{
			name:      "negative infinity",
			input:     math.Inf(-1),
			wantErr:   true,
			wantErrIs: ErrNonFiniteResult,
		},
		{name: "string", input: "abc", wantErr: true},
		{name: "nil", input: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.toDecimal(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				} else {
					assert.Contains(t, err.Error(), "cannot convert")
				}
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "expected %s, got %s", tt.want, got)
		})
	}
}

func TestValidateResultType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{name: "float64", input: float64(1.5), wantErr: false},
		{name: "int", input: int(1), wantErr: false},
		{name: "uint64", input: uint64(1), wantErr: false},
		{name: "decimal", input: decimal.NewFromInt(1), wantErr: false},
		{name: "null decimal", input: decimal.NullDecimal{}, wantErr: false},
		{name: "bool", input: true, wantErr: false},
		{name: "string", input: "hello", wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "slice", input: []any{1.0}, wantErr: true},
		{name: "map", input: map[string]any{"a": 1.0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateResultType(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrNonNumericResult)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCompileCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("same expression and env types produce same key", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("x * 2", map[string]any{"x": 1.0, "y": "a"})
		key2 := compileCacheKey("x * 2", map[string]any{"x": 99.9, "y": "b"})
		assert.Equal(t, key1, key2)
	})

	t.Run("key is a sha256 hex string", func(t *testing.T) {
		t.Parallel()
		key := compileCacheKey("x * 2", map[string]any{"x": 1.0})
		assert.Len(t, key, 64)
	})

	t.Run("different value types produce different keys", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("x", map[string]any{"x": 1.0})
		key2 := compileCacheKey("x", map[string]any{"x": ""})
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different expressions produce different keys", func(t *testing.T) {
		t.Parallel()
		env := map[string]any{"x": 1.0}
		assert.NotEqual(t, compileCacheKey("x", env), compileCacheKey("x + 1", env))
	})

	t.Run("nil and typed values produce different keys", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("x", map[string]any{"x": nil})
		key2 := compileCacheKey("x", map[string]any{"x": 1.0})
		assert.NotEqual(t, key1, key2)
	})

	t.Run("any slices of different lengths and element types produce same key", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("len(a)", map[string]any{"a": []any{1.0}})
		key2 := compileCacheKey("len(a)", map[string]any{"a": []any{"x", "y", "z"}})
		assert.Equal(t, key1, key2)
	})

	t.Run("nested map value types affect key", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("m.a", map[string]any{"m": map[string]any{"a": 1.0}})
		key2 := compileCacheKey("m.a", map[string]any{"m": map[string]any{"a": ""}})
		assert.NotEqual(t, key1, key2)
	})

	t.Run("expression is length-prefixed against env boundary collisions", func(t *testing.T) {
		t.Parallel()
		key1 := compileCacheKey("x", map[string]any{"foo": 1.0})
		key2 := compileCacheKey("x{3:foo7:float64}", map[string]any{})
		assert.NotEqual(t, key1, key2)
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

	t.Run("all nil returns nil", func(t *testing.T) {
		t.Parallel()
		got, err := coalesceFn(nil, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("first is valid", func(t *testing.T) {
		t.Parallel()
		got, err := coalesceFn("hello", nil, 42.0)
		require.NoError(t, err)
		assert.Equal(t, "hello", got)
	})
}
