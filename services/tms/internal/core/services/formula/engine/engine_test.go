package engine_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSchemaJSON = `{
	"$schema": "http://json-schema.org/draft-07/schema#",
	"$id": "test",
	"type": "object",
	"x-formula-context": {
		"entityType": "TestEntity"
	},
	"x-data-source": {
		"table": "tests",
		"preloads": []
	},
	"properties": {
		"weight": {
			"type": "number",
			"x-source": {
				"field": "Weight",
				"nullable": true
			}
		}
	}
}`

func setupEngine(t *testing.T) *engine.Engine {
	t.Helper()

	e, _ := setupEngineWithRegistry(t)

	return e
}

func setupEngineWithRegistry(t *testing.T) (*engine.Engine, *schema.Registry) {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	require.NoError(t, registry.Register("test", []byte(testSchemaJSON)))

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	e, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	return e, registry
}

func TestEngine_Compile(t *testing.T) {
	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		wantErr    bool
	}{
		{
			name:       "simple arithmetic",
			expression: "1 + 2",
			env:        map[string]any{},
			wantErr:    false,
		},
		{
			name:       "variable reference",
			expression: "baseRate * distance",
			env: map[string]any{
				"baseRate": 0.0,
				"distance": 0.0,
			},
			wantErr: false,
		},
		{
			name:       "function call",
			expression: "max(baseRate, distance)",
			env: map[string]any{
				"baseRate": 0.0,
				"distance": 0.0,
			},
			wantErr: false,
		},
		{
			name:       "conditional expression",
			expression: "hasHazmat ? hazmatFee : 0",
			env: map[string]any{
				"hasHazmat": false,
				"hazmatFee": 0.0,
			},
			wantErr: false,
		},
		{
			name:       "invalid syntax",
			expression: "baseRate * ",
			env:        map[string]any{"baseRate": 0.0},
			wantErr:    true,
		},
		{
			name:       "undefined variable",
			expression: "undefinedVar * 2",
			env:        map[string]any{},
			wantErr:    true,
		},
		{
			name:       "exceeds max node count",
			expression: "1" + strings.Repeat(" + 1", 1_100),
			env:        map[string]any{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := e.Compile(tt.expression, tt.env)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, compiled)
		})
	}
}

func TestEngine_Compile_Caching(t *testing.T) {
	e := setupEngine(t)

	env := map[string]any{"x": 0.0}
	expr := "x * 2"

	compiled1, err := e.Compile(expr, env)
	require.NoError(t, err)

	compiled2, err := e.Compile(expr, env)
	require.NoError(t, err)

	assert.Same(t, compiled1, compiled2, "should return same cached program")
}

func TestEngine_Compile_Caching_UsesEnvironmentShape(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	boolProgram, err := e.Compile("flag ? 1 : 0", map[string]any{"flag": false})
	require.NoError(t, err)

	stringProgram, err := e.Compile("flag == \"yes\" ? 1 : 0", map[string]any{"flag": ""})
	require.NoError(t, err)

	assert.NotSame(t, boolProgram, stringProgram)
}

func TestEngine_Evaluate(t *testing.T) {
	e := setupEngine(t)

	tests := []struct {
		name      string
		template  *formulatemplate.FormulaTemplate
		entity    any
		variables map[string]any
		want      decimal.Decimal
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "simple multiplication",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "baseRate * distance",
				SchemaID:   "test",
			},
			entity: struct{}{},
			variables: map[string]any{
				"baseRate": 2.5,
				"distance": 100.0,
			},
			want:    decimal.NewFromFloat(250.0),
			wantErr: false,
		},
		{
			name: "with rounding",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "round(baseRate * distance, 2)",
				SchemaID:   "test",
			},
			entity: struct{}{},
			variables: map[string]any{
				"baseRate": 2.555,
				"distance": 100.0,
			},
			want:    decimal.NewFromFloat(255.5),
			wantErr: false,
		},
		{
			name: "conditional with true",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "hasHazmat ? hazmatFee : 0",
				SchemaID:   "test",
			},
			entity: struct{}{},
			variables: map[string]any{
				"hasHazmat": true,
				"hazmatFee": 150.0,
			},
			want:    decimal.NewFromFloat(150.0),
			wantErr: false,
		},
		{
			name: "conditional with false",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "hasHazmat ? hazmatFee : 0",
				SchemaID:   "test",
			},
			entity: struct{}{},
			variables: map[string]any{
				"hasHazmat": false,
				"hazmatFee": 150.0,
			},
			want:    decimal.NewFromFloat(0),
			wantErr: false,
		},
		{
			name:      "nil template returns error",
			template:  nil,
			entity:    struct{}{},
			variables: map[string]any{},
			want:      decimal.Zero,
			wantErr:   true,
			wantErrIs: engine.ErrTemplateNil,
		},
		{
			name: "unknown schema returns error",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "baseRate * distance",
				SchemaID:   "nonexistent",
			},
			entity: struct{}{},
			variables: map[string]any{
				"baseRate": 2.5,
				"distance": 100.0,
			},
			want:      decimal.Zero,
			wantErr:   true,
			wantErrIs: engine.ErrSchemaNotFound,
		},
		{
			name: "undeclared variable shadowing schema field returns error",
			template: &formulatemplate.FormulaTemplate{
				ID:         pulid.MustNew("FT"),
				Expression: "weight * 2",
				SchemaID:   "test",
			},
			entity: struct{}{},
			variables: map[string]any{
				"weight": 10.0,
			},
			want:      decimal.Zero,
			wantErr:   true,
			wantErrIs: engine.ErrVariableShadowsField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
				Template:  tt.template,
				Entity:    tt.entity,
				Variables: tt.variables,
			})
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(
				t,
				tt.want.Equal(result.Value),
				"expected %s, got %s",
				tt.want,
				result.Value,
			)
		})
	}
}

func TestEngine_Evaluate_WithVariableDefaults(t *testing.T) {
	e := setupEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Expression: "baseRate * distance",
		SchemaID:   "test",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "baseRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.5,
			},
			{
				Name:         "distance",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 100.0,
			},
		},
	}

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template:  template,
		Entity:    struct{}{},
		Variables: map[string]any{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(150.0).Equal(result.Value))
}

func TestEngine_Evaluate_VariablesOverrideDefaults(t *testing.T) {
	e := setupEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Expression: "baseRate * distance",
		SchemaID:   "test",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name:         "baseRate",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 1.5,
			},
			{
				Name:         "distance",
				Type:         formulatypes.VariableValueTypeNumber,
				DefaultValue: 100.0,
			},
		},
	}

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   struct{}{},
		Variables: map[string]any{
			"baseRate": 2.5,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(250.0).Equal(result.Value))
}

func TestEngine_Evaluate_DeclaredVariableOverridesSchemaField(t *testing.T) {
	e := setupEngine(t)

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("FT"),
		Expression: "weight * 2",
		SchemaID:   "test",
		VariableDefinitions: []*formulatypes.VariableDefinition{
			{
				Name: "weight",
				Type: formulatypes.VariableValueTypeNumber,
			},
		},
	}

	result, err := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
		Template: template,
		Entity:   struct{}{},
		Variables: map[string]any{
			"weight": 10.0,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(20.0).Equal(result.Value))
}

func TestEngine_EvaluateExpression(t *testing.T) {
	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		entity     any
		schemaID   string
		variables  map[string]any
		want       decimal.Decimal
		wantErr    bool
		wantErrIs  error
	}{
		{
			name:       "simple expression",
			expression: "x + y",
			entity:     struct{}{},
			schemaID:   "test",
			variables: map[string]any{
				"x": 10.0,
				"y": 20.0,
			},
			want:    decimal.NewFromFloat(30.0),
			wantErr: false,
		},
		{
			name:       "complex formula",
			expression: "max(minCharge, baseRate + (distance * ratePerMile))",
			entity:     struct{}{},
			schemaID:   "test",
			variables: map[string]any{
				"minCharge":   100.0,
				"baseRate":    50.0,
				"distance":    200.0,
				"ratePerMile": 1.5,
			},
			want:    decimal.NewFromFloat(350.0),
			wantErr: false,
		},
		{
			name:       "unknown schema returns error",
			expression: "x + y",
			entity:     struct{}{},
			schemaID:   "nonexistent",
			variables: map[string]any{
				"x": 10.0,
				"y": 20.0,
			},
			want:      decimal.Zero,
			wantErr:   true,
			wantErrIs: engine.ErrSchemaNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.EvaluateExpression(
				t.Context(),
				&formulatemplatetypes.ExpressionEvaluationRequest{
					Expression: tt.expression,
					Entity:     tt.entity,
					SchemaID:   tt.schemaID,
					Variables:  tt.variables,
				},
			)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(
				t,
				tt.want.Equal(result.Value),
				"expected %s, got %s",
				tt.want,
				result.Value,
			)
		})
	}
}

func TestEngine_ValidateExpression_UsesSchemaTypes(t *testing.T) {
	t.Parallel()

	e, registry := setupEngineWithRegistry(t)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-validation-types",
		"type": "object",
		"x-formula-context": {
			"entityType": "Shipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"hasHazmat": {
				"type": "boolean",
				"x-source": {
					"computed": true,
					"function": "computeHasHazmat"
				}
			},
			"customer": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"x-source": {
							"path": "Customer.Name"
						}
					}
				}
			}
		}
	}`

	err := registry.Register("test-validation-types", []byte(schemaJSON))
	require.NoError(t, err)

	err = e.ValidateExpression(t.Context(), `hasHazmat ? 150 : 0`, "test-validation-types")
	require.NoError(t, err)

	err = e.ValidateExpression(
		t.Context(),
		`customer.name == "Acme" ? 150 : 0`,
		"test-validation-types",
	)
	require.NoError(t, err)
}

func TestEngine_ValidateExpression(t *testing.T) {
	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		schemaID   string
		wantErr    bool
		wantErrIs  error
	}{
		{
			name:       "schema not found returns error",
			expression: "x * y",
			schemaID:   "nonexistent",
			wantErr:    true,
			wantErrIs:  engine.ErrSchemaNotFound,
		},
		{
			name:       "numeric expression is valid",
			expression: "weight * 2",
			schemaID:   "test",
			wantErr:    false,
		},
		{
			name:       "string expression is invalid",
			expression: `"hello"`,
			schemaID:   "test",
			wantErr:    true,
			wantErrIs:  engine.ErrNonNumericResult,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.ValidateExpression(t.Context(), tt.expression, tt.schemaID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestEngine_ValidateExpressionWithEnv(t *testing.T) {
	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		wantErr    bool
		wantErrIs  error
	}{
		{
			name:       "valid expression",
			expression: "x + y",
			env: map[string]any{
				"x": 0.0,
				"y": 0.0,
			},
			wantErr: false,
		},
		{
			name:       "boolean result is valid",
			expression: "x > y",
			env: map[string]any{
				"x": 0.0,
				"y": 0.0,
			},
			wantErr: false,
		},
		{
			name:       "undefined variable",
			expression: "x + z",
			env: map[string]any{
				"x": 0.0,
			},
			wantErr: true,
		},
		{
			name:       "invalid syntax",
			expression: "x + ",
			env: map[string]any{
				"x": 0.0,
			},
			wantErr: true,
		},
		{
			name:       "string result is invalid",
			expression: `"hello"`,
			env:        map[string]any{},
			wantErr:    true,
			wantErrIs:  engine.ErrNonNumericResult,
		},
		{
			name:       "string concatenation is invalid",
			expression: `name + "!"`,
			env: map[string]any{
				"name": "",
			},
			wantErr:   true,
			wantErrIs: engine.ErrNonNumericResult,
		},
		{
			name:       "runtime error during trial run is tolerated",
			expression: "arr[10]",
			env: map[string]any{
				"arr": []any{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.ValidateExpressionWithEnv(t.Context(), tt.expression, tt.env)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestEngine_ClearCache(t *testing.T) {
	e := setupEngine(t)

	env := map[string]any{"x": 0.0}
	expr := "x * 2"

	compiled1, err := e.Compile(expr, env)
	require.NoError(t, err)

	e.ClearCache()

	compiled2, err := e.Compile(expr, env)
	require.NoError(t, err)

	assert.NotSame(t, compiled1, compiled2, "should create new program after cache clear")
}

func TestEngine_GetEnvironmentBuilder(t *testing.T) {
	e := setupEngine(t)

	builder := e.GetEnvironmentBuilder()
	require.NotNil(t, builder)
}

func TestEngine_EvaluateWithEnv(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	tests := []struct {
		name       string
		expression string
		env        map[string]any
		want       decimal.Decimal
		wantErr    bool
		wantErrIs  error
	}{
		{
			name:       "simple addition",
			expression: "x + y",
			env:        map[string]any{"x": 10.0, "y": 20.0},
			want:       decimal.NewFromFloat(30.0),
			wantErr:    false,
		},
		{
			name:       "with functions",
			expression: "max(a, b) * 2",
			env:        map[string]any{"a": 5.0, "b": 3.0},
			want:       decimal.NewFromFloat(10.0),
			wantErr:    false,
		},
		{
			name:       "boolean result true",
			expression: "a > b",
			env:        map[string]any{"a": 10.0, "b": 5.0},
			want:       decimal.NewFromInt(1),
			wantErr:    false,
		},
		{
			name:       "boolean result false",
			expression: "a > b",
			env:        map[string]any{"a": 3.0, "b": 5.0},
			want:       decimal.NewFromInt(0),
			wantErr:    false,
		},
		{
			name:       "integer result",
			expression: "a + b",
			env:        map[string]any{"a": 3, "b": 4},
			want:       decimal.NewFromInt(7),
			wantErr:    false,
		},
		{
			name:       "invalid expression",
			expression: "x +",
			env:        map[string]any{"x": 1.0},
			want:       decimal.Zero,
			wantErr:    true,
		},
		{
			name:       "runtime error undefined var",
			expression: "x + z",
			env:        map[string]any{"x": 1.0},
			want:       decimal.Zero,
			wantErr:    true,
		},
		{
			name:       "positive infinity result",
			expression: "x / y",
			env:        map[string]any{"x": 1.0, "y": 0.0},
			want:       decimal.Zero,
			wantErr:    true,
			wantErrIs:  engine.ErrNonFiniteResult,
		},
		{
			name:       "negative infinity result",
			expression: "x / y",
			env:        map[string]any{"x": -1.0, "y": 0.0},
			want:       decimal.Zero,
			wantErr:    true,
			wantErrIs:  engine.ErrNonFiniteResult,
		},
		{
			name:       "NaN result",
			expression: "x / y",
			env:        map[string]any{"x": 0.0, "y": 0.0},
			want:       decimal.Zero,
			wantErr:    true,
			wantErrIs:  engine.ErrNonFiniteResult,
		},
		{
			name:       "sqrt of negative errors at runtime",
			expression: "sqrt(x)",
			env:        map[string]any{"x": -1.0},
			want:       decimal.Zero,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.EvaluateWithEnv(t.Context(), tt.expression, tt.env)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(
				t,
				tt.want.Equal(result.Value),
				"expected %s, got %s",
				tt.want,
				result.Value,
			)
		})
	}
}

func TestEngine_EvaluateWithEnv_Int64Result(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"x": int64(42)}
	result, err := e.EvaluateWithEnv(t.Context(), "x", env)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(42).Equal(result.Value))
}

func TestEngine_EvaluateWithEnv_Int32Result(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"x": int32(10)}
	result, err := e.EvaluateWithEnv(t.Context(), "x", env)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(10).Equal(result.Value))
}

func TestEngine_EvaluateWithEnv_Float32Result(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"x": float32(3.5)}
	result, err := e.EvaluateWithEnv(t.Context(), "x", env)
	require.NoError(t, err)
	assert.InDelta(t, 3.5, result.Value.InexactFloat64(), 0.01)
}

func TestEngine_EvaluateWithEnv_UintResults(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	tests := []struct {
		name string
		env  map[string]any
		want decimal.Decimal
	}{
		{name: "uint", env: map[string]any{"x": uint(5)}, want: decimal.NewFromInt(5)},
		{name: "uint64", env: map[string]any{"x": uint64(10)}, want: decimal.NewFromInt(10)},
		{name: "uint32", env: map[string]any{"x": uint32(20)}, want: decimal.NewFromInt(20)},
		{name: "uint16", env: map[string]any{"x": uint16(30)}, want: decimal.NewFromInt(30)},
		{name: "uint8", env: map[string]any{"x": uint8(40)}, want: decimal.NewFromInt(40)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.EvaluateWithEnv(t.Context(), "x", tt.env)
			require.NoError(t, err)
			assert.True(
				t,
				tt.want.Equal(result.Value),
				"expected %s, got %s",
				tt.want,
				result.Value,
			)
		})
	}
}

func TestEngine_EvaluateWithEnv_NullDecimalResult(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	t.Run("valid null decimal", func(t *testing.T) {
		t.Parallel()
		env := map[string]any{
			"x": decimal.NullDecimal{Decimal: decimal.NewFromFloat(12.5), Valid: true},
		}
		result, err := e.EvaluateWithEnv(t.Context(), "x", env)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromFloat(12.5).Equal(result.Value))
	})

	t.Run("invalid null decimal", func(t *testing.T) {
		t.Parallel()
		env := map[string]any{"x": decimal.NullDecimal{}}
		_, err := e.EvaluateWithEnv(t.Context(), "x", env)
		require.ErrorIs(t, err, engine.ErrNullResult)
	})
}

func TestEngine_EvaluateWithEnv_CoalesceAllNilResult(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"a": nil, "b": nil}
	_, err := e.EvaluateWithEnv(t.Context(), "coalesce(a, b)", env)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot convert <nil> to decimal")
}

func TestEngine_EvaluateWithEnv_StringResultError(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"x": "hello"}
	_, err := e.EvaluateWithEnv(t.Context(), "x", env)
	require.Error(t, err)
}

func TestEngine_ValidateExpression_WithRegisteredSchema(t *testing.T) {
	t.Parallel()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-validate",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			},
			"distance": {
				"type": "number",
				"x-source": {
					"field": "Distance"
				}
			}
		}
	}`

	err := registry.Register("test-validate", []byte(schemaJSON))
	require.NoError(t, err)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	e, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	t.Run("valid expression with schema", func(t *testing.T) {
		t.Parallel()
		err := e.ValidateExpression(t.Context(), "weight + distance", "test-validate")
		require.NoError(t, err)
	})

	t.Run("invalid expression with schema", func(t *testing.T) {
		t.Parallel()
		err := e.ValidateExpression(t.Context(), "weight +", "test-validate")
		require.Error(t, err)
	})
}

func TestEngine_EvaluateExpression_NonNumericResult(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	_, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: "x",
			Entity:     struct{}{},
			SchemaID:   "test",
			Variables:  map[string]any{"x": "hello"},
		},
	)
	require.Error(t, err)
}

func TestEngine_EvaluateExpression_CompileError(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	_, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: "x +",
			Entity:     struct{}{},
			SchemaID:   "test",
			Variables:  map[string]any{"x": 1.0},
		},
	)
	require.Error(t, err)
}

func TestEngine_EvaluateWithEnv_DecimalResult(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"x": decimal.NewFromFloat(42.5)}
	result, err := e.EvaluateWithEnv(t.Context(), "x", env)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(42.5).Equal(result.Value))
}

func TestEngine_Evaluate_UnresolvedNonNullableField(t *testing.T) {
	t.Parallel()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()

	schemaJSON := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "test-env-err",
		"type": "object",
		"x-formula-context": {
			"entityType": "TestShipment"
		},
		"x-data-source": {
			"table": "shipments",
			"preloads": []
		},
		"properties": {
			"weight": {
				"type": "number",
				"x-source": {
					"field": "Weight"
				}
			}
		}
	}`

	err := registry.Register("test-env-err", []byte(schemaJSON))
	require.NoError(t, err)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	e, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	t.Run("undeclared variable shadowing field is rejected", func(t *testing.T) {
		t.Parallel()

		template := &formulatemplate.FormulaTemplate{
			ID:         pulid.MustNew("FT"),
			Expression: "weight * 2",
			SchemaID:   "test-env-err",
		}

		_, evalErr := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
			Template:  template,
			Entity:    struct{}{},
			Variables: map[string]any{"weight": 10.0},
		})
		require.ErrorIs(t, evalErr, engine.ErrVariableShadowsField)
	})

	t.Run("unresolved field surfaces in evaluation error", func(t *testing.T) {
		t.Parallel()

		template := &formulatemplate.FormulaTemplate{
			ID:         pulid.MustNew("FT"),
			Expression: "weight * 2",
			SchemaID:   "test-env-err",
		}

		_, evalErr := e.Evaluate(t.Context(), &formulatemplatetypes.EvaluationRequest{
			Template:  template,
			Entity:    struct{}{},
			Variables: map[string]any{},
		})
		require.Error(t, evalErr)
		require.ErrorContains(t, evalErr, "unresolved fields")
		require.ErrorContains(t, evalErr, "weight")
	})
}

func TestEngine_EvaluateWithEnv_RuntimeError(t *testing.T) {
	t.Parallel()

	e := setupEngine(t)

	env := map[string]any{"arr": []int{1, 2, 3}}
	_, err := e.EvaluateWithEnv(t.Context(), "arr[10]", env)
	require.Error(t, err)
}

func TestNewEngine(t *testing.T) {
	t.Parallel()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	e, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})

	require.NoError(t, err)
	require.NotNil(t, e)
}
