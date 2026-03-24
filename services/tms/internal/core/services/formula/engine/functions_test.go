package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileAndRun(t *testing.T, expression string, env map[string]any) any {
	t.Helper()

	options := append(engine.BuiltinFunctions(), expr.Env(env))
	program, err := expr.Compile(expression, options...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)

	return result
}

func TestRoundFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "round to nearest integer up",
			expression: "round(3.7)",
			want:       4.0,
		},
		{
			name:       "round to nearest integer down",
			expression: "round(3.2)",
			want:       3.0,
		},
		{
			name:       "round to 2 decimal places",
			expression: "round(3.14159, 2)",
			want:       3.14,
		},
		{
			name:       "round to 3 decimal places",
			expression: "round(3.14159, 3)",
			want:       3.142,
		},
		{
			name:       "round negative number",
			expression: "round(-2.7)",
			want:       -3.0,
		},
		{
			name:       "round half up",
			expression: "round(2.5)",
			want:       3.0,
		},
		{
			name:       "round zero",
			expression: "round(0.0)",
			want:       0.0,
		},
		{
			name:       "round to 0 decimal places explicitly",
			expression: "round(3.7, 0)",
			want:       4.0,
		},
		{
			name:       "round with 1 decimal place",
			expression: "round(3.456, 1)",
			want:       3.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.InDelta(t, tt.want, result, 0.001)
		})
	}
}

func TestRoundFunctionWithIntegerInput(t *testing.T) {
	t.Parallel()

	env := map[string]any{"val": 5}
	options := append(engine.BuiltinFunctions(), expr.Env(env))
	program, err := expr.Compile("round(val)", options...)
	require.NoError(t, err)

	result, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.InDelta(t, 5.0, result, 0.001)
}

func TestCeilFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "ceil positive fractional",
			expression: "ceil(3.2)",
			want:       4.0,
		},
		{
			name:       "ceil already integer",
			expression: "ceil(3.0)",
			want:       3.0,
		},
		{
			name:       "ceil negative",
			expression: "ceil(-3.7)",
			want:       -3.0,
		},
		{
			name:       "ceil small fraction",
			expression: "ceil(0.01)",
			want:       1.0,
		},
		{
			name:       "ceil zero",
			expression: "ceil(0.0)",
			want:       0.0,
		},
		{
			name:       "ceil negative small",
			expression: "ceil(-0.1)",
			want:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFloorFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "floor positive",
			expression: "floor(3.7)",
			want:       3.0,
		},
		{
			name:       "floor already integer",
			expression: "floor(3.0)",
			want:       3.0,
		},
		{
			name:       "floor negative",
			expression: "floor(-3.2)",
			want:       -4.0,
		},
		{
			name:       "floor zero",
			expression: "floor(0.0)",
			want:       0.0,
		},
		{
			name:       "floor large value",
			expression: "floor(999.999)",
			want:       999.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAbsFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "abs positive",
			expression: "abs(5.0)",
			want:       5.0,
		},
		{
			name:       "abs negative",
			expression: "abs(-5.0)",
			want:       5.0,
		},
		{
			name:       "abs zero",
			expression: "abs(0.0)",
			want:       0.0,
		},
		{
			name:       "abs large negative",
			expression: "abs(-999.99)",
			want:       999.99,
		},
		{
			name:       "abs small fraction",
			expression: "abs(-0.001)",
			want:       0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.InDelta(t, tt.want, result, 0.0001)
		})
	}
}

func TestMinFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "min first smaller",
			expression: "min(3.0, 5.0)",
			want:       3.0,
		},
		{
			name:       "min second smaller",
			expression: "min(5.0, 3.0)",
			want:       3.0,
		},
		{
			name:       "min equal values",
			expression: "min(3.0, 3.0)",
			want:       3.0,
		},
		{
			name:       "min with negative",
			expression: "min(-5.0, 3.0)",
			want:       -5.0,
		},
		{
			name:       "min both negative",
			expression: "min(-2.0, -5.0)",
			want:       -5.0,
		},
		{
			name:       "min with zero",
			expression: "min(0.0, 5.0)",
			want:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMaxFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "max first larger",
			expression: "max(5.0, 3.0)",
			want:       5.0,
		},
		{
			name:       "max second larger",
			expression: "max(3.0, 5.0)",
			want:       5.0,
		},
		{
			name:       "max equal values",
			expression: "max(3.0, 3.0)",
			want:       3.0,
		},
		{
			name:       "max with negative",
			expression: "max(-5.0, 3.0)",
			want:       3.0,
		},
		{
			name:       "max both negative",
			expression: "max(-2.0, -5.0)",
			want:       -2.0,
		},
		{
			name:       "max with zero",
			expression: "max(0.0, -5.0)",
			want:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSumFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "sum two values",
			expression: "sum(1.0, 2.0)",
			want:       3.0,
		},
		{
			name:       "sum multiple values",
			expression: "sum(1.0, 2.0, 3.0, 4.0)",
			want:       10.0,
		},
		{
			name:       "sum single value",
			expression: "sum(5.0)",
			want:       5.0,
		},
		{
			name:       "sum with negatives",
			expression: "sum(10.0, -3.0, 5.0)",
			want:       12.0,
		},
		{
			name:       "sum all negative",
			expression: "sum(-1.0, -2.0, -3.0)",
			want:       -6.0,
		},
		{
			name:       "sum with zeros",
			expression: "sum(0.0, 0.0, 5.0)",
			want:       5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAvgFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "avg two values",
			expression: "avg(2.0, 4.0)",
			want:       3.0,
		},
		{
			name:       "avg multiple values",
			expression: "avg(1.0, 2.0, 3.0, 4.0)",
			want:       2.5,
		},
		{
			name:       "avg single value",
			expression: "avg(5.0)",
			want:       5.0,
		},
		{
			name:       "avg with negatives",
			expression: "avg(-4.0, 4.0)",
			want:       0.0,
		},
		{
			name:       "avg identical values",
			expression: "avg(3.0, 3.0, 3.0)",
			want:       3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.InDelta(t, tt.want, result, 0.001)
		})
	}
}

func TestCoalesceFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       any
	}{
		{
			name:       "first non-nil",
			expression: "coalesce(a, b, c)",
			env: map[string]any{
				"a": 1.0,
				"b": 2.0,
				"c": 3.0,
			},
			want: 1.0,
		},
		{
			name:       "skip nil values",
			expression: "coalesce(a, b, c)",
			env: map[string]any{
				"a": nil,
				"b": nil,
				"c": 3.0,
			},
			want: 3.0,
		},
		{
			name:       "first value is nil",
			expression: "coalesce(a, b)",
			env: map[string]any{
				"a": nil,
				"b": "hello",
			},
			want: "hello",
		},
		{
			name:       "string result",
			expression: "coalesce(a, b)",
			env: map[string]any{
				"a": nil,
				"b": "default",
			},
			want: "default",
		},
		{
			name:       "boolean result",
			expression: "coalesce(a, b)",
			env: map[string]any{
				"a": nil,
				"b": true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, tt.env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCoalesceFunctionAllNil(t *testing.T) {
	t.Parallel()

	env := map[string]any{
		"a": nil,
		"b": nil,
	}
	options := append(engine.BuiltinFunctions(), expr.Env(env))
	program, err := expr.Compile("coalesce(a, b)", options...)
	require.NoError(t, err)

	_, err = expr.Run(program, env)
	require.Error(t, err)
}

func TestClampFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "value within range",
			expression: "clamp(5.0, 0.0, 10.0)",
			want:       5.0,
		},
		{
			name:       "value below min",
			expression: "clamp(-5.0, 0.0, 10.0)",
			want:       0.0,
		},
		{
			name:       "value above max",
			expression: "clamp(15.0, 0.0, 10.0)",
			want:       10.0,
		},
		{
			name:       "value equals min",
			expression: "clamp(0.0, 0.0, 10.0)",
			want:       0.0,
		},
		{
			name:       "value equals max",
			expression: "clamp(10.0, 0.0, 10.0)",
			want:       10.0,
		},
		{
			name:       "negative range",
			expression: "clamp(-5.0, -10.0, -1.0)",
			want:       -5.0,
		},
		{
			name:       "clamp below negative range",
			expression: "clamp(-20.0, -10.0, -1.0)",
			want:       -10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestPowFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "power of 3",
			expression: "pow(2.0, 3.0)",
			want:       8.0,
		},
		{
			name:       "power of 0",
			expression: "pow(5.0, 0.0)",
			want:       1.0,
		},
		{
			name:       "power of 1",
			expression: "pow(5.0, 1.0)",
			want:       5.0,
		},
		{
			name:       "square",
			expression: "pow(4.0, 2.0)",
			want:       16.0,
		},
		{
			name:       "fractional exponent",
			expression: "pow(9.0, 0.5)",
			want:       3.0,
		},
		{
			name:       "negative exponent",
			expression: "pow(2.0, -1.0)",
			want:       0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.InDelta(t, tt.want, result, 0.001)
		})
	}
}

func TestSqrtFunction(t *testing.T) {
	t.Parallel()

	env := map[string]any{}

	tests := []struct {
		name       string
		expression string
		want       float64
	}{
		{
			name:       "sqrt of 4",
			expression: "sqrt(4.0)",
			want:       2.0,
		},
		{
			name:       "sqrt of 9",
			expression: "sqrt(9.0)",
			want:       3.0,
		},
		{
			name:       "sqrt of 0",
			expression: "sqrt(0.0)",
			want:       0.0,
		},
		{
			name:       "sqrt of 2",
			expression: "sqrt(2.0)",
			want:       1.4142,
		},
		{
			name:       "sqrt of 16",
			expression: "sqrt(16.0)",
			want:       4.0,
		},
		{
			name:       "sqrt of 100",
			expression: "sqrt(100.0)",
			want:       10.0,
		},
		{
			name:       "sqrt of 1",
			expression: "sqrt(1.0)",
			want:       1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, env)
			assert.InDelta(t, tt.want, result, 0.001)
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       float64
	}{
		{
			name:       "billing formula with max",
			expression: "max(baseRate, ratePerMile * distance)",
			env: map[string]any{
				"baseRate":    100.0,
				"ratePerMile": 2.5,
				"distance":    50.0,
			},
			want: 125.0,
		},
		{
			name:       "billing formula base rate wins",
			expression: "max(baseRate, ratePerMile * distance)",
			env: map[string]any{
				"baseRate":    200.0,
				"ratePerMile": 2.5,
				"distance":    50.0,
			},
			want: 200.0,
		},
		{
			name:       "weight-based pricing with rounding",
			expression: "round(ceil(weight / 100) * ratePerCWT, 2)",
			env: map[string]any{
				"weight":     4550.0,
				"ratePerCWT": 15.75,
			},
			want: 724.5,
		},
		{
			name:       "conditional hazmat fee true",
			expression: "hasHazmat ? hazmatFee : 0.0",
			env: map[string]any{
				"hasHazmat": true,
				"hazmatFee": 150.0,
			},
			want: 150.0,
		},
		{
			name:       "conditional hazmat fee false",
			expression: "hasHazmat ? hazmatFee : 0.0",
			env: map[string]any{
				"hasHazmat": false,
				"hazmatFee": 150.0,
			},
			want: 0.0,
		},
		{
			name:       "combined formula with clamp",
			expression: "clamp(baseRate + (distance * ratePerMile), minCharge, maxCharge)",
			env: map[string]any{
				"baseRate":    50.0,
				"distance":    100.0,
				"ratePerMile": 1.5,
				"minCharge":   100.0,
				"maxCharge":   500.0,
			},
			want: 200.0,
		},
		{
			name:       "sum with avg",
			expression: "sum(avg(a, b), avg(c, d))",
			env: map[string]any{
				"a": 10.0,
				"b": 20.0,
				"c": 30.0,
				"d": 40.0,
			},
			want: 50.0,
		},
		{
			name:       "nested function calls",
			expression: "round(pow(sqrt(x), 2.0), 2)",
			env: map[string]any{
				"x": 7.0,
			},
			want: 7.0,
		},
		{
			name:       "abs with min",
			expression: "min(abs(a), abs(b))",
			env: map[string]any{
				"a": -10.0,
				"b": 5.0,
			},
			want: 5.0,
		},
		{
			name:       "floor division with clamp",
			expression: "clamp(floor(total / units), 1.0, 100.0)",
			env: map[string]any{
				"total": 550.0,
				"units": 6.0,
			},
			want: 91.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := compileAndRun(t, tt.expression, tt.env)
			assert.InDelta(t, tt.want, result, 0.01)
		})
	}
}

func TestFunctionsWithIntegerInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       float64
	}{
		{
			name:       "ceil with int",
			expression: "ceil(x)",
			env:        map[string]any{"x": 5},
			want:       5.0,
		},
		{
			name:       "floor with int",
			expression: "floor(x)",
			env:        map[string]any{"x": 5},
			want:       5.0,
		},
		{
			name:       "abs with int",
			expression: "abs(x)",
			env:        map[string]any{"x": -5},
			want:       5.0,
		},
		{
			name:       "min with int",
			expression: "min(a, b)",
			env:        map[string]any{"a": 3, "b": 7},
			want:       3.0,
		},
		{
			name:       "max with int",
			expression: "max(a, b)",
			env:        map[string]any{"a": 3, "b": 7},
			want:       7.0,
		},
		{
			name:       "pow with int",
			expression: "pow(a, b)",
			env:        map[string]any{"a": 2, "b": 3},
			want:       8.0,
		},
		{
			name:       "sqrt with int",
			expression: "sqrt(x)",
			env:        map[string]any{"x": 25},
			want:       5.0,
		},
		{
			name:       "sum with int",
			expression: "sum(a, b, c)",
			env:        map[string]any{"a": 1, "b": 2, "c": 3},
			want:       6.0,
		},
		{
			name:       "avg with int",
			expression: "avg(a, b)",
			env:        map[string]any{"a": 4, "b": 6},
			want:       5.0,
		},
		{
			name:       "clamp with int",
			expression: "clamp(x, lo, hi)",
			env:        map[string]any{"x": 15, "lo": 0, "hi": 10},
			want:       10.0,
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

func TestFunctionsWithInt64Inputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       float64
	}{
		{
			name:       "ceil with int64",
			expression: "ceil(x)",
			env:        map[string]any{"x": int64(5)},
			want:       5.0,
		},
		{
			name:       "abs with int64",
			expression: "abs(x)",
			env:        map[string]any{"x": int64(-10)},
			want:       10.0,
		},
		{
			name:       "sqrt with int64",
			expression: "sqrt(x)",
			env:        map[string]any{"x": int64(49)},
			want:       7.0,
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

func TestFunctionsWithInt32Inputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       float64
	}{
		{
			name:       "abs with int32",
			expression: "abs(x)",
			env:        map[string]any{"x": int32(-7)},
			want:       7.0,
		},
		{
			name:       "sqrt with int32",
			expression: "sqrt(x)",
			env:        map[string]any{"x": int32(36)},
			want:       6.0,
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

func TestBuiltinFunctions(t *testing.T) {
	t.Parallel()

	funcs := engine.BuiltinFunctions()
	assert.NotEmpty(t, funcs)
	assert.Len(t, funcs, 12)
}
