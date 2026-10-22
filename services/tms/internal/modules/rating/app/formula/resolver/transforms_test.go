package resolver_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransform(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{"decimal", true},
		{"integer", true},
		{"decimalToFloat64", false},
		{"stringToUpper", true},
		{"stringToLower", true},
		{"unixToISO8601", true},
		{"nonExistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := resolver.GetTransform(tt.name)
			assert.Equal(t, tt.exists, ok)
			if tt.exists {
				assert.NotNil(t, fn)
			}
		})
	}
}

func TestRegisterTransform(t *testing.T) {
	customTransform := func(value any) (any, error) {
		return "custom", nil
	}

	resolver.RegisterTransform("customTransform", customTransform)

	fn, ok := resolver.GetTransform("customTransform")
	require.True(t, ok)
	require.NotNil(t, fn)

	result, err := fn(nil)
	require.NoError(t, err)
	assert.Equal(t, "custom", result)
}

func TestExactDecimal_KeepsTheDecimalAndReadsAnEmptyOneAsNil(t *testing.T) {
	fn, ok := resolver.GetTransform("decimal")
	require.True(t, ok)

	value := decimal.RequireFromString("123.456")

	got, err := fn(value)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	got, err = fn(&value)
	require.NoError(t, err)
	assert.Equal(t, value, got)

	got, err = fn(decimal.NewNullDecimal(decimal.RequireFromString("99.99")))
	require.NoError(t, err)
	assert.Equal(t, decimal.RequireFromString("99.99"), got)

	for _, empty := range []any{(*decimal.Decimal)(nil), decimal.NullDecimal{}, (*decimal.NullDecimal)(nil)} {
		got, err = fn(empty)
		require.NoError(t, err)
		assert.Nil(t, got, "%T", empty)
	}

	_, err = fn("not a decimal")
	require.Error(t, err)
}

func TestExactInteger_WidensToInt64AndReadsAnEmptyOneAsNil(t *testing.T) {
	fn, ok := resolver.GetTransform("integer")
	require.True(t, ok)

	i64, i, i16, i32 := int64(42), 100, int16(-7), int32(9)
	for value, want := range map[any]int64{i64: 42, i: 100, i16: -7, i32: 9} {
		got, err := fn(value)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for value, want := range map[any]int64{&i64: 42, &i: 100, &i16: -7, &i32: 9} {
		got, err := fn(value)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	for _, empty := range []any{(*int64)(nil), (*int)(nil), (*int16)(nil), (*int32)(nil)} {
		got, err := fn(empty)
		require.NoError(t, err)
		assert.Nil(t, got, "%T", empty)
	}

	_, err := fn(1.5)
	require.Error(t, err, "a float is not a whole number")
}

func TestStringToUpper(t *testing.T) {
	fn, ok := resolver.GetTransform("stringToUpper")
	require.True(t, ok)

	s := "hello"

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "string value",
			value:   "hello world",
			want:    "HELLO WORLD",
			wantErr: false,
		},
		{
			name:    "string pointer",
			value:   &s,
			want:    "HELLO",
			wantErr: false,
		},
		{
			name:    "nil string pointer",
			value:   (*string)(nil),
			want:    "",
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   123,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringToLower(t *testing.T) {
	fn, ok := resolver.GetTransform("stringToLower")
	require.True(t, ok)

	s := "HELLO"

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "string value",
			value:   "HELLO WORLD",
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "string pointer",
			value:   &s,
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "nil string pointer",
			value:   (*string)(nil),
			want:    "",
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   123,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnixToISO8601(t *testing.T) {
	fn, ok := resolver.GetTransform("unixToISO8601")
	require.True(t, ok)

	ts := int64(1609459200)

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "int64 value",
			value:   int64(1609459200),
			want:    time.Unix(1609459200, 0).UTC().Format(time.RFC3339),
			wantErr: false,
		},
		{
			name:    "int64 pointer",
			value:   &ts,
			want:    time.Unix(1609459200, 0).UTC().Format(time.RFC3339),
			wantErr: false,
		},
		{
			name:    "nil int64 pointer",
			value:   (*int64)(nil),
			want:    "",
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   "not a timestamp",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
