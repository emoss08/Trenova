package engine_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const datetimeSchemaJSON = `{
	"$id": "dated",
	"type": "object",
	"properties": {
		"pickupDate": {
			"description": "When the pickup happens",
			"type": ["string", "null"],
			"format": "date-time",
			"x-source": {"computed": true, "function": "computePickupDate", "requires": ["Moves"]}
		},
		"weight": {"type": "number", "x-source": {"field": "Weight"}}
	}
}`

func datedEngine(t *testing.T) (*engine.Engine, *engine.EnvironmentBuilder) {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)
	require.NoError(t, registry.Register("dated", []byte(datetimeSchemaJSON)))

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

func TestValidationEnvironment_DatetimePropertyIsADate(t *testing.T) {
	t.Parallel()

	_, envBuilder := datedEngine(t)

	env, _, err := envBuilder.BuildValidationEnvironment("dated", nil)
	require.NoError(t, err)

	_, isTime := env["pickupDate"].(time.Time)
	assert.True(t, isTime, "a datetime property compiles as a date, got %T", env["pickupDate"])
}

func TestValidationEnvironment_CoercesStringSamplesToDates(t *testing.T) {
	t.Parallel()

	_, envBuilder := datedEngine(t)

	for _, sample := range []string{"2026-06-05T20:30:00Z", "2026-06-05T20:30", "2026-06-05"} {
		env, _, err := envBuilder.BuildValidationEnvironment(
			"dated",
			map[string]any{"pickupDate": sample},
		)
		require.NoError(t, err)

		parsed, isTime := env["pickupDate"].(time.Time)
		require.True(t, isTime, "sample %q should become a date, got %T", sample, env["pickupDate"])
		assert.Equal(t, 2026, parsed.Year(), sample)
		assert.Equal(t, time.June, parsed.Month(), sample)
	}
}

func TestEngine_DatetimeVariablesEvaluateWithExprDateMethods(t *testing.T) {
	t.Parallel()

	e, _ := datedEngine(t)

	result, err := e.EvaluateWithEnv(
		t.Context(),
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: `pickupDate.Hour() >= 18 ? 75 : 0`,
			Env: map[string]any{
				"pickupDate": time.Date(2026, time.June, 5, 20, 30, 0, 0, time.UTC),
				"weight":     100.0,
			},
			Lookup: engine.StubLookup{},
		},
	)

	require.NoError(t, err)
	assert.True(t, result.Value.Equal(decimal.NewFromInt(75)))
}
