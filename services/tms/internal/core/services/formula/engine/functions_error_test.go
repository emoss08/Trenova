package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileAndRunExpectError(t *testing.T, expression string, env map[string]any) {
	t.Helper()

	options := append(engine.BuiltinFunctions(), expr.Env(env))
	program, err := expr.Compile(expression, options...)
	if err != nil {
		return
	}

	_, err = expr.Run(program, env)
	require.Error(t, err)
}

func TestToFloat64_InvalidTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string argument to ceil",
			expression: "ceil(x)",
			env:        map[string]any{"x": "notanumber"},
		},
		{
			name:       "bool argument to floor",
			expression: "floor(x)",
			env:        map[string]any{"x": true},
		},
		{
			name:       "slice argument to abs",
			expression: "abs(x)",
			env:        map[string]any{"x": []int{1, 2, 3}},
		},
		{
			name:       "map argument to sqrt",
			expression: "sqrt(x)",
			env:        map[string]any{"x": map[string]int{"a": 1}},
		},
		{
			name:       "nil argument to ceil",
			expression: "ceil(x)",
			env:        map[string]any{"x": nil},
		},
		{
			name:       "struct argument to floor",
			expression: "floor(x)",
			env:        map[string]any{"x": struct{ V int }{V: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestToInt_InvalidTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string argument to round decimals",
			expression: "round(3.14, x)",
			env:        map[string]any{"x": "two"},
		},
		{
			name:       "bool argument to round decimals",
			expression: "round(3.14, x)",
			env:        map[string]any{"x": true},
		},
		{
			name:       "slice argument to round decimals",
			expression: "round(3.14, x)",
			env:        map[string]any{"x": []int{2}},
		},
		{
			name:       "nil argument to round decimals",
			expression: "round(3.14, x)",
			env:        map[string]any{"x": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestCeilFn_ErrorBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string input",
			expression: "ceil(x)",
			env:        map[string]any{"x": "abc"},
		},
		{
			name:       "bool input",
			expression: "ceil(x)",
			env:        map[string]any{"x": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestFloorFn_ErrorBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string input",
			expression: "floor(x)",
			env:        map[string]any{"x": "abc"},
		},
		{
			name:       "map input",
			expression: "floor(x)",
			env:        map[string]any{"x": map[string]string{"k": "v"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestAbsFn_ErrorBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string input",
			expression: "abs(x)",
			env:        map[string]any{"x": "negative"},
		},
		{
			name:       "bool input",
			expression: "abs(x)",
			env:        map[string]any{"x": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestSqrtFn_ErrorBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string input",
			expression: "sqrt(x)",
			env:        map[string]any{"x": "nine"},
		},
		{
			name:       "bool input",
			expression: "sqrt(x)",
			env:        map[string]any{"x": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestPowFn_ErrorBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string base",
			expression: "pow(x, 2.0)",
			env:        map[string]any{"x": "two"},
		},
		{
			name:       "string exponent",
			expression: "pow(2.0, x)",
			env:        map[string]any{"x": "three"},
		},
		{
			name:       "both invalid",
			expression: "pow(a, b)",
			env:        map[string]any{"a": "x", "b": "y"},
		},
		{
			name:       "bool base",
			expression: "pow(x, 2.0)",
			env:        map[string]any{"x": true},
		},
		{
			name:       "bool exponent",
			expression: "pow(2.0, x)",
			env:        map[string]any{"x": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestMinFn_ErrorBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string first argument",
			expression: "min(a, 5.0)",
			env:        map[string]any{"a": "abc"},
		},
		{
			name:       "string second argument",
			expression: "min(5.0, b)",
			env:        map[string]any{"b": "xyz"},
		},
		{
			name:       "both string",
			expression: "min(a, b)",
			env:        map[string]any{"a": "x", "b": "y"},
		},
		{
			name:       "bool first argument",
			expression: "min(a, 5.0)",
			env:        map[string]any{"a": true},
		},
		{
			name:       "bool second argument",
			expression: "min(5.0, b)",
			env:        map[string]any{"b": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestMaxFn_ErrorBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string first argument",
			expression: "max(a, 5.0)",
			env:        map[string]any{"a": "abc"},
		},
		{
			name:       "string second argument",
			expression: "max(5.0, b)",
			env:        map[string]any{"b": "xyz"},
		},
		{
			name:       "both string",
			expression: "max(a, b)",
			env:        map[string]any{"a": "x", "b": "y"},
		},
		{
			name:       "bool first argument",
			expression: "max(a, 5.0)",
			env:        map[string]any{"a": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestClampFn_ErrorBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string value argument",
			expression: "clamp(x, 0.0, 10.0)",
			env:        map[string]any{"x": "five"},
		},
		{
			name:       "string min argument",
			expression: "clamp(5.0, lo, 10.0)",
			env:        map[string]any{"lo": "zero"},
		},
		{
			name:       "string max argument",
			expression: "clamp(5.0, 0.0, hi)",
			env:        map[string]any{"hi": "ten"},
		},
		{
			name:       "bool value argument",
			expression: "clamp(x, 0.0, 10.0)",
			env:        map[string]any{"x": true},
		},
		{
			name:       "bool min argument",
			expression: "clamp(5.0, lo, 10.0)",
			env:        map[string]any{"lo": false},
		},
		{
			name:       "bool max argument",
			expression: "clamp(5.0, 0.0, hi)",
			env:        map[string]any{"hi": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestAvgFn_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("avg with invalid type in arguments", func(t *testing.T) {
		t.Parallel()
		compileAndRunExpectError(t, "avg(a, b)", map[string]any{"a": "x", "b": "y"})
	})

	t.Run("avg with single value", func(t *testing.T) {
		t.Parallel()
		env := map[string]any{"x": 7.0}
		options := append(engine.BuiltinFunctions(), expr.Env(env))
		program, err := expr.Compile("avg(x)", options...)
		require.NoError(t, err)
		result, err := expr.Run(program, env)
		require.NoError(t, err)
		assert.InDelta(t, 7.0, result, 0.001)
	})
}

func TestSumFn_ErrorBranch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string in sum",
			expression: "sum(a, b)",
			env:        map[string]any{"a": "hello", "b": 2.0},
		},
		{
			name:       "bool in sum",
			expression: "sum(a, b)",
			env:        map[string]any{"a": 1.0, "b": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestRoundFn_ErrorBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
	}{
		{
			name:       "string value argument",
			expression: "round(x)",
			env:        map[string]any{"x": "abc"},
		},
		{
			name:       "bool value argument",
			expression: "round(x)",
			env:        map[string]any{"x": true},
		},
		{
			name:       "string value with decimals",
			expression: "round(x, 2)",
			env:        map[string]any{"x": "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			compileAndRunExpectError(t, tt.expression, tt.env)
		})
	}
}

func TestToInt_WithValidTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       float64
	}{
		{
			name:       "int64 decimals arg",
			expression: "round(3.14159, x)",
			env:        map[string]any{"x": int64(2)},
			want:       3.14,
		},
		{
			name:       "int32 decimals arg",
			expression: "round(3.14159, x)",
			env:        map[string]any{"x": int32(3)},
			want:       3.142,
		},
		{
			name:       "int decimals arg",
			expression: "round(3.14159, x)",
			env:        map[string]any{"x": 1},
			want:       3.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, tt.env)
			assert.InDelta(t, tt.want, result, 0.001)
		})
	}
}
