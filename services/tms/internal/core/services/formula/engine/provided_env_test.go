package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const providedSchemaJSON = `{
	"$id": "fueled",
	"type": "object",
	"properties": {
		"fuelPrice": {
			"description": "Latest diesel price per gallon from the tenant's fuel index",
			"type": ["number", "null"],
			"x-source": {"provided": true, "nullable": true}
		},
		"weight": {"type": "number", "x-source": {"field": "Weight"}}
	}
}`

type fueledEntity struct {
	Weight float64
}

func fueledEngine(t *testing.T) (*engine.Engine, *engine.EnvironmentBuilder) {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	require.NoError(t, registry.Register("fueled", []byte(providedSchemaJSON)))

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
	return e, envBuilder
}

func TestBuild_ProvidedPropertyIsAPlaceholderNotAFailure(t *testing.T) {
	t.Parallel()

	_, envBuilder := fueledEngine(t)

	env, failures, err := envBuilder.Build(&fueledEntity{Weight: 10}, "fueled")
	require.NoError(t, err)

	value, present := env["fuelPrice"]
	assert.True(t, present, "the variable exists so expressions compile")
	assert.Nil(t, value)
	assert.NotContains(t, failures, "fuelPrice", "nothing on the record could have supplied it")
}

func TestBuildWithProvided_ProvidedValuesLandBeneathCallerVariables(t *testing.T) {
	t.Parallel()

	_, envBuilder := fueledEngine(t)

	env, _, err := envBuilder.BuildWithProvided(
		&fueledEntity{Weight: 10},
		"fueled",
		map[string]any{"fuelPrice": 3.85},
		nil,
	)
	require.NoError(t, err)
	assert.InDelta(t, 3.85, env["fuelPrice"], 0.0001)

	env, _, err = envBuilder.BuildWithProvided(
		&fueledEntity{Weight: 10},
		"fueled",
		map[string]any{"fuelPrice": 3.85},
		map[string]any{"fuelPrice": 4.10},
	)
	require.NoError(t, err)
	assert.InDelta(t, 4.10, env["fuelPrice"], 0.0001, "a caller's variable wins over the feed")

	validation, _, err := envBuilder.BuildValidationEnvironmentWithProvided(
		"fueled",
		map[string]any{"fuelPrice": 3.85},
		nil,
	)
	require.NoError(t, err)
	assert.InDelta(t, 3.85, validation["fuelPrice"], 0.0001, "previews see the feed too")
}

func TestEngine_ProvidedVariablesAreTracedAsProvided(t *testing.T) {
	t.Parallel()

	e, _ := fueledEngine(t)

	result, err := e.EvaluateExpression(
		t.Context(),
		&formulatemplatetypes.ExpressionEvaluationRequest{
			Expression: "weight * fuelPrice",
			Entity:     &fueledEntity{Weight: 10},
			SchemaID:   "fueled",
			Provided:   map[string]any{"fuelPrice": 3.85},
			Lookup:     engine.StubLookup{},
		},
	)
	require.NoError(t, err)
	assert.InDelta(t, 38.5, result.Value.InexactFloat64(), 0.0001)

	require.NotNil(t, result.Receipt)
	var source formulatypes.ValueSource
	for _, variable := range result.Receipt.Variables {
		if variable.Name == "fuelPrice" {
			source = variable.Source
		}
	}
	assert.Equal(t, formulatypes.ValueSourceProvided, source)
}
