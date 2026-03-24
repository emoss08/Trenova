package resolver_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransform(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{"decimalToFloat64", true},
		{"int64ToFloat64", true},
		{"int16ToFloat64", true},
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

func TestDecimalToFloat64(t *testing.T) {
	fn, ok := resolver.GetTransform("decimalToFloat64")
	require.True(t, ok)

	d := decimal.NewFromFloat(123.456)
	ptrD := &d

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "decimal value",
			value:   decimal.NewFromFloat(123.456),
			want:    123.456,
			wantErr: false,
		},
		{
			name:    "decimal pointer",
			value:   ptrD,
			want:    123.456,
			wantErr: false,
		},
		{
			name:    "nil decimal pointer",
			value:   (*decimal.Decimal)(nil),
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "NullDecimal valid",
			value:   decimal.NullDecimal{Decimal: decimal.NewFromFloat(99.99), Valid: true},
			want:    99.99,
			wantErr: false,
		},
		{
			name:    "NullDecimal invalid",
			value:   decimal.NullDecimal{Valid: false},
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   "not a decimal",
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
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestInt64ToFloat64(t *testing.T) {
	fn, ok := resolver.GetTransform("int64ToFloat64")
	require.True(t, ok)

	i64 := int64(42)
	i := 100

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "int64 value",
			value:   int64(42),
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "int64 pointer",
			value:   &i64,
			want:    42.0,
			wantErr: false,
		},
		{
			name:    "nil int64 pointer",
			value:   (*int64)(nil),
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "int value",
			value:   100,
			want:    100.0,
			wantErr: false,
		},
		{
			name:    "int pointer",
			value:   &i,
			want:    100.0,
			wantErr: false,
		},
		{
			name:    "nil int pointer",
			value:   (*int)(nil),
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   "not an int",
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

func TestInt16ToFloat64(t *testing.T) {
	fn, ok := resolver.GetTransform("int16ToFloat64")
	require.True(t, ok)

	i16 := int16(16)

	tests := []struct {
		name    string
		value   any
		want    any
		wantErr bool
	}{
		{
			name:    "int16 value",
			value:   int16(16),
			want:    16.0,
			wantErr: false,
		},
		{
			name:    "int16 pointer",
			value:   &i16,
			want:    16.0,
			wantErr: false,
		},
		{
			name:    "nil int16 pointer",
			value:   (*int16)(nil),
			want:    0.0,
			wantErr: false,
		},
		{
			name:    "unsupported type",
			value:   int32(32),
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
