// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package expression

import (
	"math"
	"testing"
)

func TestEvaluateBinaryOp(t *testing.T) {
	tests := []struct {
		name      string
		op        TokenType
		left      any
		right     any
		want      any
		wantError bool
	}{
		// Arithmetic operations
		{
			name:  "add numbers",
			op:    TokenPlus,
			left:  5.0,
			right: 3.0,
			want:  8.0,
		},
		{
			name:  "add integers",
			op:    TokenPlus,
			left:  10,
			right: 20,
			want:  30.0,
		},
		{
			name:  "string concatenation left",
			op:    TokenPlus,
			left:  "hello",
			right: " world",
			want:  "hello world",
		},
		{
			name:  "string concatenation right",
			op:    TokenPlus,
			left:  42,
			right: " answer",
			want:  "42 answer",
		},
		{
			name:  "subtract numbers",
			op:    TokenMinus,
			left:  10.0,
			right: 3.0,
			want:  7.0,
		},
		{
			name:  "multiply numbers",
			op:    TokenMultiply,
			left:  4.0,
			right: 5.0,
			want:  20.0,
		},
		{
			name:  "divide numbers",
			op:    TokenDivide,
			left:  20.0,
			right: 4.0,
			want:  5.0,
		},
		{
			name:      "divide by zero",
			op:        TokenDivide,
			left:      10.0,
			right:     0.0,
			wantError: true,
		},
		{
			name:  "modulo",
			op:    TokenModulo,
			left:  10.0,
			right: 3.0,
			want:  1.0,
		},
		{
			name:      "modulo by zero",
			op:        TokenModulo,
			left:      10.0,
			right:     0.0,
			wantError: true,
		},
		{
			name:  "power",
			op:    TokenPower,
			left:  2.0,
			right: 3.0,
			want:  8.0,
		},

		// Comparison operations
		{
			name:  "equal true",
			op:    TokenEqual,
			left:  5.0,
			right: 5.0,
			want:  true,
		},
		{
			name:  "equal false",
			op:    TokenEqual,
			left:  5.0,
			right: 6.0,
			want:  false,
		},
		{
			name:  "equal strings",
			op:    TokenEqual,
			left:  "test",
			right: "test",
			want:  true,
		},
		{
			name:  "not equal true",
			op:    TokenNotEqual,
			left:  5.0,
			right: 6.0,
			want:  true,
		},
		{
			name:  "greater true",
			op:    TokenGreater,
			left:  10.0,
			right: 5.0,
			want:  true,
		},
		{
			name:  "greater false",
			op:    TokenGreater,
			left:  5.0,
			right: 10.0,
			want:  false,
		},
		{
			name:  "less true",
			op:    TokenLess,
			left:  5.0,
			right: 10.0,
			want:  true,
		},
		{
			name:  "greater equal true",
			op:    TokenGreaterEqual,
			left:  10.0,
			right: 10.0,
			want:  true,
		},
		{
			name:  "less equal true",
			op:    TokenLessEqual,
			left:  5.0,
			right: 5.0,
			want:  true,
		},

		// Logical operations
		{
			name:  "and true true",
			op:    TokenAnd,
			left:  true,
			right: true,
			want:  true,
		},
		{
			name:  "and true false",
			op:    TokenAnd,
			left:  true,
			right: false,
			want:  false,
		},
		{
			name:  "or true false",
			op:    TokenOr,
			left:  true,
			right: false,
			want:  true,
		},
		{
			name:  "or false false",
			op:    TokenOr,
			left:  false,
			right: false,
			want:  false,
		},

		// Type coercion
		{
			name:  "add mixed types",
			op:    TokenPlus,
			left:  5.5,
			right: int(2),
			want:  7.5,
		},
		{
			name:      "subtract non-numeric",
			op:        TokenMinus,
			left:      "hello",
			right:     5,
			wantError: true,
		},

		// Overflow checks
		{
			name:      "add overflow",
			op:        TokenPlus,
			left:      math.MaxFloat64,
			right:     math.MaxFloat64,
			wantError: true,
		},
		{
			name:      "multiply overflow",
			op:        TokenMultiply,
			left:      math.MaxFloat64,
			right:     2.0,
			wantError: true,
		},
		{
			name:      "power overflow",
			op:        TokenPower,
			left:      1000.0,
			right:     1000.0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateBinaryOp(tt.op, tt.left, tt.right)

			if (err != nil) != tt.wantError {
				t.Errorf("evaluateBinaryOp() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Compare results
			switch want := tt.want.(type) {
			case float64:
				got, ok := got.(float64)
				if !ok {
					t.Errorf("evaluateBinaryOp() = %T, want float64", got)
					return
				}
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("evaluateBinaryOp() = %v, want %v", got, want)
				}
			case bool:
				if got != want {
					t.Errorf("evaluateBinaryOp() = %v, want %v", got, want)
				}
			case string:
				if got != want {
					t.Errorf("evaluateBinaryOp() = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestEvaluateUnaryOp(t *testing.T) {
	tests := []struct {
		name      string
		op        TokenType
		operand   any
		want      any
		wantError bool
	}{
		{
			name:    "negate positive",
			op:      TokenMinus,
			operand: 5.0,
			want:    -5.0,
		},
		{
			name:    "negate negative",
			op:      TokenMinus,
			operand: -3.0,
			want:    3.0,
		},
		{
			name:    "negate integer",
			op:      TokenMinus,
			operand: 10,
			want:    -10.0,
		},
		{
			name:      "negate string",
			op:        TokenMinus,
			operand:   "hello",
			wantError: true,
		},
		{
			name:    "not true",
			op:      TokenNot,
			operand: true,
			want:    false,
		},
		{
			name:    "not false",
			op:      TokenNot,
			operand: false,
			want:    true,
		},
		{
			name:    "not number zero",
			op:      TokenNot,
			operand: 0.0,
			want:    true,
		},
		{
			name:    "not number non-zero",
			op:      TokenNot,
			operand: 5.0,
			want:    false,
		},
		{
			name:    "not empty string",
			op:      TokenNot,
			operand: "",
			want:    true,
		},
		{
			name:    "not non-empty string",
			op:      TokenNot,
			operand: "hello",
			want:    false,
		},
		{
			name:    "not nil",
			op:      TokenNot,
			operand: nil,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateUnaryOp(tt.op, tt.operand)

			if (err != nil) != tt.wantError {
				t.Errorf("evaluateUnaryOp() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Compare results
			switch want := tt.want.(type) {
			case float64:
				got, ok := got.(float64)
				if !ok {
					t.Errorf("evaluateUnaryOp() = %T, want float64", got)
					return
				}
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("evaluateUnaryOp() = %v, want %v", got, want)
				}
			case bool:
				if got != want {
					t.Errorf("evaluateUnaryOp() = %v, want %v", got, want)
				}
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"number zero", 0.0, false},
		{"number positive", 5.0, true},
		{"number negative", -5.0, true},
		{"string empty", "", false},
		{"string non-empty", "hello", true},
		{"array empty", []any{}, false},
		{"array non-empty", []any{1, 2, 3}, true},
		{"nil", nil, false},
		{"other type", struct{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toBool(tt.value); got != tt.want {
				t.Errorf("toBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualValues(t *testing.T) {
	tests := []struct {
		name  string
		left  any
		right any
		want  bool
	}{
		// Numbers
		{"equal numbers", 5.0, 5.0, true},
		{"different numbers", 5.0, 6.0, false},
		{"float epsilon", 1.0000000001, 1.0000000002, true}, // Within epsilon
		{"mixed numeric types", float64(5), int(5), true},

		// Strings
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},

		// Booleans
		{"equal bools true", true, true, true},
		{"equal bools false", false, false, true},
		{"different bools", true, false, false},

		// Arrays
		{"equal arrays", []any{1, 2, 3}, []any{1, 2, 3}, true},
		{"different arrays length", []any{1, 2}, []any{1, 2, 3}, false},
		{"different arrays content", []any{1, 2, 3}, []any{1, 2, 4}, false},
		{"empty arrays", []any{}, []any{}, true},

		// Nil
		{"both nil", nil, nil, true},
		{"left nil", nil, 5, false},
		{"right nil", "hello", nil, false},

		// Mixed types
		{"number string", 5.0, "5", false},
		{"bool number", true, 1.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := equalValues(tt.left, tt.right); got != tt.want {
				t.Errorf("equalValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name  string
		left  any
		right any
		want  int
	}{
		// Numbers
		{"less than", 5.0, 10.0, -1},
		{"greater than", 10.0, 5.0, 1},
		{"equal", 5.0, 5.0, 0},
		{"mixed types less", int(5), 10.0, -1},

		// Strings
		{"string less", "apple", "banana", -1},
		{"string greater", "zebra", "apple", 1},
		{"string equal", "hello", "hello", 0},

		// Incomparable types
		{"number string", 5.0, "hello", 0},
		{"bool number", true, 5.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareValues(tt.left, tt.right); got != tt.want {
				t.Errorf("compareValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
