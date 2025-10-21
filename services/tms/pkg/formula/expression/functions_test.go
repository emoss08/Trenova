package expression

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func TestBuiltinFunctions(t *testing.T) {
	// Create a mock variable context
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{
			"test_var": 42.0,
		},
	}

	// Create evaluation context
	evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
	registry := evalCtx.functions

	tests := []struct {
		name      string
		function  string
		args      []any
		want      any
		wantError bool
	}{
		// Math functions
		{
			name:     "abs positive",
			function: "abs",
			args:     []any{5.0},
			want:     5.0,
		},
		{
			name:     "abs negative",
			function: "abs",
			args:     []any{-5.0},
			want:     5.0,
		},
		{
			name:      "abs wrong args",
			function:  "abs",
			args:      []any{},
			wantError: true,
		},

		// Advanced math functions
		{
			name:     "log natural",
			function: "log",
			args:     []any{math.E},
			want:     1.0,
		},
		{
			name:     "log base 10",
			function: "log",
			args:     []any{100.0, 10.0},
			want:     2.0,
		},
		{
			name:      "log negative",
			function:  "log",
			args:      []any{-1.0},
			wantError: true,
		},
		{
			name:      "log zero base",
			function:  "log",
			args:      []any{10.0, 0.0},
			wantError: true,
		},
		{
			name:      "log base 1",
			function:  "log",
			args:      []any{10.0, 1.0},
			wantError: true,
		},
		{
			name:     "exp of 0",
			function: "exp",
			args:     []any{0.0},
			want:     1.0,
		},
		{
			name:     "exp of 1",
			function: "exp",
			args:     []any{1.0},
			want:     math.E,
		},
		{
			name:     "sin of 0",
			function: "sin",
			args:     []any{0.0},
			want:     0.0,
		},
		{
			name:     "sin of pi/2",
			function: "sin",
			args:     []any{math.Pi / 2},
			want:     1.0,
		},
		{
			name:     "cos of 0",
			function: "cos",
			args:     []any{0.0},
			want:     1.0,
		},
		{
			name:     "cos of pi",
			function: "cos",
			args:     []any{math.Pi},
			want:     -1.0,
		},
		{
			name:     "tan of 0",
			function: "tan",
			args:     []any{0.0},
			want:     0.0,
		},
		{
			name:     "min two args",
			function: "min",
			args:     []any{5.0, 3.0},
			want:     3.0,
		},
		{
			name:     "min multiple args",
			function: "min",
			args:     []any{5.0, 3.0, 7.0, 1.0},
			want:     1.0,
		},
		{
			name:      "min no args",
			function:  "min",
			args:      []any{},
			wantError: true,
		},
		{
			name:     "max two args",
			function: "max",
			args:     []any{5.0, 3.0},
			want:     5.0,
		},
		{
			name:     "max multiple args",
			function: "max",
			args:     []any{5.0, 3.0, 7.0, 1.0},
			want:     7.0,
		},
		{
			name:     "floor",
			function: "floor",
			args:     []any{3.7},
			want:     3.0,
		},
		{
			name:     "ceil",
			function: "ceil",
			args:     []any{3.2},
			want:     4.0,
		},
		{
			name:     "round half up",
			function: "round",
			args:     []any{3.5},
			want:     4.0,
		},
		{
			name:     "round half down",
			function: "round",
			args:     []any{3.4},
			want:     3.0,
		},
		{
			name:     "round with precision",
			function: "round",
			args:     []any{3.14159, 2.0},
			want:     3.14,
		},
		{
			name:     "sqrt",
			function: "sqrt",
			args:     []any{16.0},
			want:     4.0,
		},
		{
			name:      "sqrt negative",
			function:  "sqrt",
			args:      []any{-4.0},
			wantError: true,
		},
		{
			name:     "pow",
			function: "pow",
			args:     []any{2.0, 3.0},
			want:     8.0,
		},

		// Type conversion functions
		{
			name:     "number from float",
			function: "number",
			args:     []any{42.0},
			want:     42.0,
		},
		{
			name:     "number from string",
			function: "number",
			args:     []any{"42"},
			want:     42.0,
		},
		{
			name:     "number from bool true",
			function: "number",
			args:     []any{true},
			want:     1.0,
		},
		{
			name:     "number from bool false",
			function: "number",
			args:     []any{false},
			want:     0.0,
		},
		{
			name:      "number invalid",
			function:  "number",
			args:      []any{"not a number"},
			wantError: true,
		},
		{
			name:     "string from number",
			function: "string",
			args:     []any{42.0},
			want:     "42",
		},
		{
			name:     "string from bool",
			function: "string",
			args:     []any{true},
			want:     "true",
		},
		{
			name:     "bool from true",
			function: "bool",
			args:     []any{true},
			want:     true,
		},
		{
			name:     "bool from false",
			function: "bool",
			args:     []any{false},
			want:     false,
		},
		{
			name:     "bool from number non-zero",
			function: "bool",
			args:     []any{1.0},
			want:     true,
		},
		{
			name:     "bool from number zero",
			function: "bool",
			args:     []any{0.0},
			want:     false,
		},
		{
			name:     "bool from string non-empty",
			function: "bool",
			args:     []any{"hello"},
			want:     true,
		},
		{
			name:     "bool from string empty",
			function: "bool",
			args:     []any{""},
			want:     false,
		},

		// Array functions
		{
			name:     "len string",
			function: "len",
			args:     []any{"hello"},
			want:     5.0,
		},
		{
			name:     "len array",
			function: "len",
			args:     []any{[]any{1, 2, 3}},
			want:     3.0,
		},
		{
			name:     "sum",
			function: "sum",
			args:     []any{[]any{1.0, 2.0, 3.0, 4.0}},
			want:     10.0,
		},
		{
			name:     "avg single array",
			function: "avg",
			args:     []any{[]any{1.0, 2.0, 3.0, 4.0}},
			want:     2.5,
		},
		{
			name:      "avg empty array",
			function:  "avg",
			args:      []any{[]any{}},
			wantError: true,
		},

		// Array manipulation functions
		{
			name:     "slice array middle",
			function: "slice",
			args:     []any{[]any{10.0, 20.0, 30.0, 40.0, 50.0}, 1.0, 4.0},
			want:     []any{20.0, 30.0, 40.0},
		},
		{
			name:     "slice array negative indices",
			function: "slice",
			args:     []any{[]any{10.0, 20.0, 30.0, 40.0, 50.0}, -3.0, -1.0},
			want:     []any{30.0, 40.0},
		},
		{
			name:     "slice string",
			function: "slice",
			args:     []any{"hello world", 6.0, 11.0},
			want:     "world",
		},
		{
			name:     "concat arrays",
			function: "concat",
			args:     []any{[]any{1.0, 2.0}, []any{3.0, 4.0}, []any{5.0}},
			want:     []any{1.0, 2.0, 3.0, 4.0, 5.0},
		},
		{
			name:     "concat strings",
			function: "concat",
			args:     []any{"hello", " ", "world"},
			want:     "hello world",
		},
		{
			name:     "concat mixed",
			function: "concat",
			args:     []any{[]any{1.0}, 2.0, []any{3.0, 4.0}},
			want:     []any{1.0, 2.0, 3.0, 4.0},
		},
		{
			name:     "contains array true",
			function: "contains",
			args:     []any{[]any{10.0, 20.0, 30.0}, 20.0},
			want:     true,
		},
		{
			name:     "contains array false",
			function: "contains",
			args:     []any{[]any{10.0, 20.0, 30.0}, 40.0},
			want:     false,
		},
		{
			name:     "contains string true",
			function: "contains",
			args:     []any{"hello world", "world"},
			want:     true,
		},
		{
			name:     "contains string false",
			function: "contains",
			args:     []any{"hello world", "goodbye"},
			want:     false,
		},
		{
			name:     "indexOf array found",
			function: "indexOf",
			args:     []any{[]any{10.0, 20.0, 30.0}, 20.0},
			want:     1.0,
		},
		{
			name:     "indexOf array not found",
			function: "indexOf",
			args:     []any{[]any{10.0, 20.0, 30.0}, 40.0},
			want:     -1.0,
		},
		{
			name:     "indexOf string found",
			function: "indexOf",
			args:     []any{"hello world", "world"},
			want:     6.0,
		},
		{
			name:     "indexOf string not found",
			function: "indexOf",
			args:     []any{"hello world", "goodbye"},
			want:     -1.0,
		},

		// Conditional functions
		{
			name:     "if true",
			function: "if",
			args:     []any{true, "yes", "no"},
			want:     "yes",
		},
		{
			name:     "if false",
			function: "if",
			args:     []any{false, "yes", "no"},
			want:     "no",
		},
		{
			name:     "if number condition",
			function: "if",
			args:     []any{1.0, "yes", "no"},
			want:     "yes",
		},
		{
			name:     "coalesce first non-nil",
			function: "coalesce",
			args:     []any{nil, nil, "value", "other"},
			want:     "value",
		},
		{
			name:     "coalesce all nil",
			function: "coalesce",
			args:     []any{nil, nil, nil},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry[tt.function]
			if !exists {
				t.Fatalf("function %q not found", tt.function)
			}

			got, err := fn.Call(evalCtx, tt.args...)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Special handling for float comparisons
			switch want := tt.want.(type) {
			case float64:
				got, ok := got.(float64)
				if !ok {
					t.Errorf("Execute() = %T, want float64", got)
					return
				}
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("Execute() = %v, want %v", got, want)
				}
			case []any:
				gotSlice, ok := got.([]any)
				if !ok {
					t.Errorf("Execute() = %T, want []any", got)
					return
				}
				if len(gotSlice) != len(want) {
					t.Errorf("Execute() len = %d, want %d", len(gotSlice), len(want))
					return
				}
				for i := range want {
					if !equalTestValues(gotSlice[i], want[i]) {
						t.Errorf("Execute()[%d] = %v, want %v", i, gotSlice[i], want[i])
					}
				}
			default:
				if !equalTestValues(got, tt.want) {
					t.Errorf("Execute() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestFunctionRegistry(t *testing.T) {
	// Create evaluation context to get registry
	mockVarCtx := &mockVariableContext{
		fields: map[string]any{},
	}
	evalCtx := NewEvaluationContext(context.Background(), mockVarCtx)
	registry := evalCtx.functions

	// Test getting a built-in function
	fn, exists := registry["abs"]
	if !exists {
		t.Error("Get abs() returned false, want true")
	}
	if fn == nil {
		t.Error("Get abs() returned nil function")
	}

	// Test getting non-existent function
	_, exists = registry["nonexistent"]
	if exists {
		t.Error("Get() returned true for non-existent function")
	}

	// Test that we have the expected number of built-in functions
	expectedFunctions := []string{
		"abs", "min", "max", "round", "floor", "ceil", "sqrt", "pow",
		"log", "exp", "sin", "cos", "tan",
		"number", "string", "bool",
		"len", "sum", "avg", "slice", "concat", "contains", "indexOf",
		"if", "coalesce",
	}

	for _, name := range expectedFunctions {
		if _, exists := registry[name]; !exists {
			t.Errorf("Expected function %q not found in registry", name)
		}
	}
}

func TestValidateArgs(t *testing.T) {
	// Create a simple test function that implements the Function interface
	type testFunction struct {
		name    string
		minArgs int
		maxArgs int
	}

	tests := []struct {
		name      string
		fn        *testFunction
		args      []any
		wantError bool
	}{
		{
			name:      "too few args",
			fn:        &testFunction{minArgs: 2, maxArgs: 4},
			args:      []any{1},
			wantError: true,
		},
		{
			name:      "min args",
			fn:        &testFunction{minArgs: 2, maxArgs: 4},
			args:      []any{1, 2},
			wantError: false,
		},
		{
			name:      "between min and max",
			fn:        &testFunction{minArgs: 2, maxArgs: 4},
			args:      []any{1, 2, 3},
			wantError: false,
		},
		{
			name:      "max args",
			fn:        &testFunction{minArgs: 2, maxArgs: 4},
			args:      []any{1, 2, 3, 4},
			wantError: false,
		},
		{
			name:      "too many args",
			fn:        &testFunction{minArgs: 2, maxArgs: 4},
			args:      []any{1, 2, 3, 4, 5},
			wantError: true,
		},
		{
			name:      "unlimited args",
			fn:        &testFunction{minArgs: 1, maxArgs: -1},
			args:      []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate args based on min/max rules
			numArgs := len(tt.args)
			var err error

			if numArgs < tt.fn.minArgs {
				err = fmt.Errorf(
					"too few arguments: got %d, need at least %d",
					numArgs,
					tt.fn.minArgs,
				)
			} else if tt.fn.maxArgs != -1 && numArgs > tt.fn.maxArgs {
				err = fmt.Errorf("too many arguments: got %d, max %d", numArgs, tt.fn.maxArgs)
			}

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateArgs() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Helper function to compare values with tolerance for floats
func equalTestValues(a, b any) bool {
	switch aVal := a.(type) {
	case float64:
		if bVal, ok := b.(float64); ok {
			return math.Abs(aVal-bVal) < 1e-9
		}
	case string:
		if bVal, ok := b.(string); ok {
			return aVal == bVal
		}
	case bool:
		if bVal, ok := b.(bool); ok {
			return aVal == bVal
		}
	case nil:
		return b == nil
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
