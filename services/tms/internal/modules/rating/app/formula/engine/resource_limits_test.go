package engine_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLimitedEngine(t *testing.T, slots int) *engine.Engine {
	t.Helper()

	registry := schema.NewRegistry()
	res := resolver.NewResolver()
	e, err := engine.NewEngine(engine.Params{
		Registry: registry,
		Resolver: res,
		EnvBuilder: engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
			Registry: registry,
			Resolver: res,
		}),
		Config: &config.Config{
			Formula: config.FormulaConfig{MaxConcurrentEvaluations: slots},
		},
	})
	require.NoError(t, err)

	return e
}

func evaluateEnv(
	ctx context.Context,
	e *engine.Engine,
	expression string,
	env map[string]any,
) (*formulatemplatetypes.EvaluationResult, error) {
	if env == nil {
		env = map[string]any{}
	}
	return e.EvaluateWithEnv(ctx, &formulatemplatetypes.EnvEvaluationRequest{
		Expression: expression,
		Env:        env,
	})
}

func TestEvaluate_AnAbandonedEvaluationHoldsItsSlotUntilItFinishes(t *testing.T) {
	t.Parallel()

	e := newLimitedEngine(t, 1)
	release := make(chan struct{})
	blocked := map[string]any{
		"block": func() float64 {
			<-release
			return 1
		},
	}

	short := engine.WithEvaluationTimeout(t.Context(), 50*time.Millisecond)

	_, err := evaluateEnv(short, e, `block() + 1`, blocked)
	require.ErrorIs(t, err, engine.ErrEvaluationTimeout)

	_, err = evaluateEnv(short, e, `1 + 1`, nil)
	require.ErrorIs(t, err, engine.ErrEvaluationTimeout,
		"the abandoned evaluation still runs, so it still holds the only slot")

	close(release)

	require.Eventually(t, func() bool {
		result, evalErr := evaluateEnv(short, e, `1 + 1`, nil)
		return evalErr == nil && result.Value.IntPart() == 2
	}, 2*time.Second, 10*time.Millisecond)
}

func TestEvaluate_TheVMMemoryBudgetIsBounded(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)

	result, err := evaluateEnv(t.Context(), e, `len(1..n)`, map[string]any{"n": 1_000})
	require.NoError(t, err)
	assert.EqualValues(t, 1_000, result.Value.IntPart())

	_, err = evaluateEnv(t.Context(), e, `len(1..n)`, map[string]any{"n": 150_000})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory budget exceeded")
}

func TestEvaluate_StringGrowthIsBounded(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)
	env := map[string]any{"code": strings.Repeat("x", 100)}

	cases := map[string]string{
		"concatenation doubling": `len(reduce(1..22, #acc + #acc, "xxxxxxxx"))`,
		"concatenation in a map": `len(map(1..20000, code + "-"))`,
		"replace doubling":       `len(reduce(1..22, replace(#acc, "x", "xx"), "x"))`,
		"join":                   `len(join(map(1..20000, code), ""))`,
		"string of a list":       `len(string(map(1..20000, code)))`,
	}

	for name, expression := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := evaluateEnv(t.Context(), e, expression, env)
			require.ErrorIs(t, err, engine.ErrStringBudgetExceeded)
		})
	}
}

func TestEvaluate_StringsWithinTheBudgetStillWork(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)
	env := map[string]any{
		"origin":      map[string]any{"state": "TX"},
		"destination": "CA",
		"miles":       120.0,
	}

	result, err := evaluateEnv(t.Context(), e,
		`origin.state + "-" + destination == "TX-CA" ? miles + 1 : 0`, env)
	require.NoError(t, err)
	assert.EqualValues(t, 121, result.Value.IntPart())

	result, err = evaluateEnv(t.Context(), e,
		`len(join(["a", "b"], "-") + replace("ab", "b", "cd") + string(12))`, env)
	require.NoError(t, err)
	assert.EqualValues(t, 8, result.Value.IntPart())

	_, err = evaluateEnv(t.Context(), e, `"lane" + 5`, env)
	require.Error(t, err, "a typed string plus a number stays a compile error")
}
