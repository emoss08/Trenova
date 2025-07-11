package expression

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

func TestEvaluator_Evaluate(t *testing.T) {
	// Create mock variable context
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{
			"distance":       100.0,
			"weight":         500.0,
			"base_rate":      2.5,
			"has_hazmat":     true,
			"temperature":    -10.0,
			"equipment_type": "Reefer",
			"prices":         []any{10.5, 20.0, 30.75},
			"matrix":         []any{[]any{1.0, 2.0}, []any{3.0, 4.0}},
			"index":          1.0,
		},
	}

	tests := []struct {
		name      string
		expr      string
		want      float64
		wantError bool
	}{
		// Literals
		{
			name: "number literal",
			expr: "42",
			want: 42.0,
		},
		{
			name: "string literal",
			expr: `"hello"`,
			want: 0, // Strings would return 0
			wantError: true, // Expression must return numeric
		},

		// Arithmetic
		{
			name: "simple addition",
			expr: "2 + 3",
			want: 5.0,
		},
		{
			name: "multiplication precedence",
			expr: "2 + 3 * 4",
			want: 14.0,
		},
		{
			name: "parentheses",
			expr: "(2 + 3) * 4",
			want: 20.0,
		},
		{
			name: "power operation",
			expr: "2 ^ 3",
			want: 8.0,
		},
		{
			name: "complex arithmetic",
			expr: "10 - 3 * 2 + 8 / 4",
			want: 6.0,
		},

		// Comparison (returns 1 for true, 0 for false)
		{
			name: "greater than true",
			expr: "5 > 3",
			want: 1.0,
		},
		{
			name: "less than false",
			expr: "5 < 3",
			want: 0.0,
		},
		{
			name: "equal numbers",
			expr: "3.14 == 3.14",
			want: 1.0,
		},
		{
			name: "not equal strings",
			expr: `"hello" != "world"`,
			want: 1.0, // String comparison works, returns true (1.0)
		},

		// Logical (returns 1 for true, 0 for false)
		{
			name: "logical and true", 
			expr: "1 && 1",  // Use 1 for true
			want: 1.0,
		},
		{
			name: "logical or mixed",
			expr: "1 || 0",  // Use 1 for true, 0 for false
			want: 1.0,
		},
		{
			name: "logical not",
			expr: "!0",  // Use 0 for false
			want: 1.0,
		},
		{
			name: "complex logical",
			expr: "(5 > 3) && (2 < 4) || false",
			want: 1.0,
		},

		// Ternary
		{
			name: "ternary true",
			expr: "true ? 10 : 20",
			want: 10.0,
		},
		{
			name: "ternary false",
			expr: "false ? 10 : 20",
			want: 20.0,
		},
		{
			name: "ternary with expression",
			expr: "5 > 3 ? 100 : 200",
			want: 100.0,
		},

		// Functions
		{
			name: "abs function",
			expr: "abs(-42)",
			want: 42.0,
		},
		{
			name: "min function",
			expr: "min(5, 3, 7, 1)",
			want: 1.0,
		},
		{
			name: "max function",
			expr: "max(5, 3, 7, 1)",
			want: 7.0,
		},
		{
			name: "round function",
			expr: "round(3.14159, 2)",
			want: 3.14,
		},
		{
			name: "nested functions",
			expr: "max(abs(-5), min(10, 20))",
			want: 10.0,
		},

		// Array operations
		{
			name: "array literal",
			expr: "[1, 2, 3]",
			want: 0.0,
			wantError: true, // Arrays not numeric
		},
		{
			name: "array indexing",
			expr: "[10, 20, 30][1]",
			want: 20.0,
		},
		{
			name: "variable array indexing",
			expr: "prices[1]",
			want: 20.0,
			wantError: true, // Variables not registered
		},
		{
			name: "nested array indexing",
			expr: "[[1, 2], [3, 4]][1][0]",
			want: 3.0,
		},
		{
			name: "computed index",
			expr: "[10, 20, 30][1 + 1]",
			want: 30.0,
		},
		{
			name: "string indexing",
			expr: `"hello"[1]`,
			want: 0.0,
			wantError: true, // Would return "e" which is not numeric
		},

		// String concatenation (would fail - expressions must return numeric)
		{
			name: "string concat",
			expr: `"hello" + " " + "world"`,
			want: 0.0,
			wantError: true,
		},
		{
			name: "mixed concat",
			expr: `"Value: " + 42`,
			want: 0.0,
			wantError: true,
		},

		// Variables (would need to be registered)
		{
			name: "variable reference",
			expr: "distance * base_rate",
			want: 250.0,
			wantError: true, // Variables not registered in this test
		},

		// Error cases
		{
			name:      "divide by zero",
			expr:      "10 / 0",
			wantError: true,
		},
		{
			name:      "unknown function",
			expr:      "unknown(42)",
			wantError: true,
		},
		{
			name:      "syntax error",
			expr:      "2 + + 3",
			wantError: true,
		},
		{
			name:      "unclosed paren",
			expr:      "(2 + 3",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create variable registry
			varRegistry := variables.NewRegistry()
			
			// Create evaluator
			evaluator := NewEvaluator(varRegistry)
			
			// Evaluate expression
			got, err := evaluator.Evaluate(context.Background(), tt.expr, mockVarCtx)
			
			if (err != nil) != tt.wantError {
				t.Errorf("Evaluate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			
			if tt.wantError {
				return
			}
			
			// Compare results
			if !equalFloatValues(got, tt.want) {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluator_ParseOnly(t *testing.T) {
	// Create variable registry
	varRegistry := variables.NewRegistry()
	evaluator := NewEvaluator(varRegistry)
	
	tests := []struct {
		name      string
		expr      string
		wantError bool
	}{
		{
			name: "valid expression",
			expr: "2 + 3 * 4",
		},
		{
			name: "complex expression",
			expr: "max(a, b) > 10 && c < 20 ? d * 2 : e / 3",
		},
		{
			name:      "syntax error",
			expr:      "2 + + 3",
			wantError: true,
		},
		{
			name:      "empty expression",
			expr:      "",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test by compiling the expression
			compiled, err := evaluator.compile(tt.expr)
			
			if (err != nil) != tt.wantError {
				t.Errorf("compile() error = %v, wantError %v", err, tt.wantError)
				return
			}
			
			if tt.wantError {
				return
			}
			
			if compiled == nil || compiled.ast == nil {
				t.Error("compile() returned nil AST")
			}
		})
	}
}

func TestEvaluator_Caching(t *testing.T) {
	varRegistry := variables.NewRegistry()
	evaluator := NewEvaluator(varRegistry)
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{},
	}
	
	// Evaluate same expression multiple times
	expr := "2 + 3 * 4"
	
	for i := 0; i < 5; i++ {
		result, err := evaluator.Evaluate(context.Background(), expr, mockVarCtx)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if result != 14.0 {
			t.Errorf("Evaluate() = %v, want 14.0", result)
		}
	}
	
	// The AST should be cached after first parse
	// (This is more of an implementation detail test)
}

func TestEvaluator_ComplexExpressions(t *testing.T) {
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{},
	}
	
	tests := []struct {
		name      string
		expr      string
		want      float64
		wantError bool
	}{
		{
			name: "nested conditionals",
			expr: "true ? (false ? 1 : 2) : 3",
			want: 2.0,
		},
		{
			name: "multiple operators",
			expr: "1 + 2 * 3 - 4 / 2 + 5 % 3",
			want: 7.0,
		},
		{
			name: "function in conditional",
			expr: "max(5, 10) > 7 ? abs(-20) : min(30, 40)",
			want: 20.0,
		},
		{
			name:      "string operations",
			expr:      `"Result: " + (10 > 5 ? "YES" : "NO")`,
			want:      0.0,
			wantError: true, // This would fail - can't return string
		},
	}
	
	varRegistry := variables.NewRegistry()
	evaluator := NewEvaluator(varRegistry)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(context.Background(), tt.expr, mockVarCtx)
			
			if (err != nil) != tt.wantError {
				t.Errorf("Evaluate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			
			if tt.wantError {
				return
			}
			
			if !equalFloatValues(got, tt.want) {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkEvaluator(b *testing.B) {
	varRegistry := variables.NewRegistry()
	evaluator := NewEvaluator(varRegistry)
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{},
	}
	
	expressions := []string{
		"2 + 3",
		"2 + 3 * 4 - 5 / 2",
		"max(10, 20) + min(5, 3)",
		"true ? 100 : 200",
		"(5 > 3) && (10 < 20) || false",
	}
	
	for _, expr := range expressions {
		b.Run(expr, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := evaluator.Evaluate(context.Background(), expr, mockVarCtx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Helper function to compare values with tolerance for floats
func equalFloatValues(a, b float64) bool {
	return a == b || (a-b < 0.0001 && b-a < 0.0001)
}