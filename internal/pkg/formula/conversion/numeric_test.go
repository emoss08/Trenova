/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package conversion_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/conversion"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		ok       bool
	}{
		// Nil handling
		{
			name:     "nil value",
			input:    nil,
			expected: 0,
			ok:       false,
		},

		// Direct float types
		{
			name:     "float64",
			input:    float64(42.5),
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "float32",
			input:    float32(42.5),
			expected: 42.5,
			ok:       true,
		},

		// Integer types
		{
			name:     "int",
			input:    int(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "int8",
			input:    int8(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "int16",
			input:    int16(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "int32",
			input:    int32(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "int64",
			input:    int64(42),
			expected: 42.0,
			ok:       true,
		},

		// Unsigned integer types
		{
			name:     "uint",
			input:    uint(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "uint8",
			input:    uint8(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "uint16",
			input:    uint16(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "uint32",
			input:    uint32(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "uint64",
			input:    uint64(42),
			expected: 42.0,
			ok:       true,
		},

		// Pointer types - non-nil
		{
			name:     "non-nil float64 pointer",
			input:    float64Ptr(42.5),
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "non-nil float32 pointer",
			input:    float32Ptr(42.5),
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "non-nil int pointer",
			input:    intPtr(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "non-nil int16 pointer",
			input:    int16Ptr(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "non-nil int32 pointer",
			input:    int32Ptr(42),
			expected: 42.0,
			ok:       true,
		},
		{
			name:     "non-nil int64 pointer",
			input:    int64Ptr(42),
			expected: 42.0,
			ok:       true,
		},

		// Pointer types - nil
		{
			name:     "nil float64 pointer",
			input:    (*float64)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil float32 pointer",
			input:    (*float32)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int pointer",
			input:    (*int)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int16 pointer",
			input:    (*int16)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int32 pointer",
			input:    (*int32)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int64 pointer",
			input:    (*int64)(nil),
			expected: 0,
			ok:       false,
		},

		// Decimal types
		{
			name:     "decimal.Decimal",
			input:    decimal.NewFromFloat(42.5),
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "decimal.Decimal with high precision",
			input:    decimal.NewFromFloat(42.123456789),
			expected: 42.123456789,
			ok:       true,
		},
		{
			name:     "valid decimal.NullDecimal",
			input:    decimal.NullDecimal{Decimal: decimal.NewFromFloat(42.5), Valid: true},
			expected: 42.5,
			ok:       true,
		},
		{
			name:     "invalid decimal.NullDecimal",
			input:    decimal.NullDecimal{Decimal: decimal.NewFromFloat(42.5), Valid: false},
			expected: 0,
			ok:       false,
		},

		// Edge cases
		{
			name:     "zero value",
			input:    int(0),
			expected: 0.0,
			ok:       true,
		},
		{
			name:     "negative value",
			input:    int(-42),
			expected: -42.0,
			ok:       true,
		},
		{
			name:     "max int64",
			input:    int64(math.MaxInt64),
			expected: float64(math.MaxInt64),
			ok:       true,
		},
		{
			name:     "min int64",
			input:    int64(math.MinInt64),
			expected: float64(math.MinInt64),
			ok:       true,
		},

		// Unsupported types
		{
			name:     "string",
			input:    "42.5",
			expected: 0,
			ok:       false,
		},
		{
			name:     "bool",
			input:    true,
			expected: 0,
			ok:       false,
		},
		{
			name:     "struct",
			input:    struct{}{},
			expected: 0,
			ok:       false,
		},
		{
			name:     "slice",
			input:    []int{1, 2, 3},
			expected: 0,
			ok:       false,
		},
		{
			name:     "map",
			input:    map[string]int{"a": 1},
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := conversion.ToFloat64(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.InDelta(t, tt.expected, result, 0.0000001)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestToFloat64OrZero(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{
			name:     "valid conversion",
			input:    int(42),
			expected: 42.0,
		},
		{
			name:     "invalid conversion returns zero",
			input:    "not a number",
			expected: 0.0,
		},
		{
			name:     "nil returns zero",
			input:    nil,
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := conversion.ToFloat64OrZero(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
		ok       bool
	}{
		// Nil handling
		{
			name:     "nil value",
			input:    nil,
			expected: 0,
			ok:       false,
		},

		// Integer types
		{
			name:     "int64",
			input:    int64(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "int",
			input:    int(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "int8",
			input:    int8(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "int16",
			input:    int16(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "int32",
			input:    int32(42),
			expected: 42,
			ok:       true,
		},

		// Unsigned types
		{
			name:     "uint",
			input:    uint(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "uint8",
			input:    uint8(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "uint16",
			input:    uint16(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "uint32",
			input:    uint32(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "uint64 within range",
			input:    uint64(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "uint64 overflow",
			input:    uint64(math.MaxInt64) + 1,
			expected: 0,
			ok:       false,
		},

		// Float types (truncation)
		{
			name:     "float32",
			input:    float32(42.9),
			expected: 42,
			ok:       true,
		},
		{
			name:     "float64",
			input:    float64(42.9),
			expected: 42,
			ok:       true,
		},

		// Pointer types - non-nil
		{
			name:     "non-nil int64 pointer",
			input:    int64Ptr(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "non-nil int pointer",
			input:    intPtr(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "non-nil int16 pointer",
			input:    int16Ptr(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "non-nil int32 pointer",
			input:    int32Ptr(42),
			expected: 42,
			ok:       true,
		},

		// Pointer types - nil
		{
			name:     "nil int64 pointer",
			input:    (*int64)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int pointer",
			input:    (*int)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int16 pointer",
			input:    (*int16)(nil),
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil int32 pointer",
			input:    (*int32)(nil),
			expected: 0,
			ok:       false,
		},

		// Decimal types
		{
			name:     "decimal.Decimal",
			input:    decimal.NewFromInt(42),
			expected: 42,
			ok:       true,
		},
		{
			name:     "decimal.Decimal with fraction (truncated)",
			input:    decimal.NewFromFloat(42.9),
			expected: 42,
			ok:       true,
		},
		{
			name:     "valid decimal.NullDecimal",
			input:    decimal.NullDecimal{Decimal: decimal.NewFromInt(42), Valid: true},
			expected: 42,
			ok:       true,
		},
		{
			name:     "invalid decimal.NullDecimal",
			input:    decimal.NullDecimal{Decimal: decimal.NewFromInt(42), Valid: false},
			expected: 0,
			ok:       false,
		},

		// Edge cases
		{
			name:     "zero",
			input:    int64(0),
			expected: 0,
			ok:       true,
		},
		{
			name:     "negative",
			input:    int64(-42),
			expected: -42,
			ok:       true,
		},
		{
			name:     "max int64",
			input:    int64(math.MaxInt64),
			expected: math.MaxInt64,
			ok:       true,
		},
		{
			name:     "min int64",
			input:    int64(math.MinInt64),
			expected: math.MinInt64,
			ok:       true,
		},

		// Unsupported types
		{
			name:     "string",
			input:    "42",
			expected: 0,
			ok:       false,
		},
		{
			name:     "bool",
			input:    true,
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := conversion.ToInt64(tt.input)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
		ok       bool
	}{
		// Nil handling
		{
			name:     "nil value",
			input:    nil,
			expected: false,
			ok:       false,
		},

		// Boolean types
		{
			name:     "bool true",
			input:    true,
			expected: true,
			ok:       true,
		},
		{
			name:     "bool false",
			input:    false,
			expected: false,
			ok:       true,
		},
		{
			name:     "non-nil bool pointer true",
			input:    boolPtr(true),
			expected: true,
			ok:       true,
		},
		{
			name:     "non-nil bool pointer false",
			input:    boolPtr(false),
			expected: false,
			ok:       true,
		},
		{
			name:     "nil bool pointer",
			input:    (*bool)(nil),
			expected: false,
			ok:       false,
		},

		// Numeric types (0 = false, non-zero = true)
		{
			name:     "int zero",
			input:    int(0),
			expected: false,
			ok:       true,
		},
		{
			name:     "int non-zero positive",
			input:    int(42),
			expected: true,
			ok:       true,
		},
		{
			name:     "int non-zero negative",
			input:    int(-42),
			expected: true,
			ok:       true,
		},
		{
			name:     "int64 zero",
			input:    int64(0),
			expected: false,
			ok:       true,
		},
		{
			name:     "int64 non-zero",
			input:    int64(1),
			expected: true,
			ok:       true,
		},
		{
			name:     "uint zero",
			input:    uint(0),
			expected: false,
			ok:       true,
		},
		{
			name:     "uint non-zero",
			input:    uint(1),
			expected: true,
			ok:       true,
		},
		{
			name:     "float64 zero",
			input:    float64(0.0),
			expected: false,
			ok:       true,
		},
		{
			name:     "float64 non-zero",
			input:    float64(0.1),
			expected: true,
			ok:       true,
		},
		{
			name:     "float64 negative",
			input:    float64(-0.1),
			expected: true,
			ok:       true,
		},

		// String types (empty = false, non-empty = true)
		{
			name:     "empty string",
			input:    "",
			expected: false,
			ok:       true,
		},
		{
			name:     "non-empty string",
			input:    "hello",
			expected: true,
			ok:       true,
		},
		{
			name:     "string with spaces",
			input:    " ",
			expected: true,
			ok:       true,
		},

		// Unsupported types
		{
			name:     "slice",
			input:    []int{1, 2, 3},
			expected: false,
			ok:       false,
		},
		{
			name:     "map",
			input:    map[string]int{"a": 1},
			expected: false,
			ok:       false,
		},
		{
			name:     "struct",
			input:    struct{}{},
			expected: false,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := conversion.ToBool(tt.input)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions for creating pointers
func float64Ptr(v float64) *float64 { return &v }
func float32Ptr(v float32) *float32 { return &v }
func intPtr(v int) *int             { return &v }
func int16Ptr(v int16) *int16       { return &v }
func int32Ptr(v int32) *int32       { return &v }
func int64Ptr(v int64) *int64       { return &v }
func boolPtr(v bool) *bool          { return &v }
