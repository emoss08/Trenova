package engine_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBareEngine(t *testing.T) *engine.Engine {
	t.Helper()

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

	return e
}

// TestEvaluateWithEnv_RunawayExpressionIsCancelled guards the WithContext
// wiring: a loop over a billion elements fits MaxNodes easily, so without the
// injected cancellation checks the vm would pin a core for minutes after the
// caller has given up.
func TestEvaluateWithEnv_RunawayExpressionIsCancelled(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)

	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()

	before := runtime.NumGoroutine()
	start := time.Now()

	_, err := e.EvaluateWithEnv(
		ctx,
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: "all(1..1000000000, # >= 0)",
			Env:        map[string]any{},
		},
	)

	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "evaluation must stop at cancellation")

	assert.Eventually(t, func() bool {
		return runtime.NumGoroutine() <= before+2
	}, 5*time.Second, 50*time.Millisecond, "the evaluation goroutine must exit after cancellation")
}

// TestCompileCacheStableAcrossContextTypes guards the cache-key special case:
// the injected context differs in concrete type per call, and hashing it by
// type would give every request its own cache entry, silencing the LRU.
func TestCompileCacheStableAcrossContextTypes(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)
	e.ClearCache()

	env := map[string]any{"weight": 10.0}

	_, err := e.EvaluateWithEnv(
		t.Context(),
		&formulatemplatetypes.EnvEvaluationRequest{Expression: "weight * 2", Env: env},
	)
	require.NoError(t, err)

	cancelable, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, err = e.EvaluateWithEnv(
		cancelable,
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: "weight * 2",
			Env:        map[string]any{"weight": 10.0},
		},
	)
	require.NoError(t, err)

	assert.Equal(t, 1, e.CacheLen())
}

func TestValidateExpressionDetailed(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)

	t.Run("runtime error against synthetic env is a warning", func(t *testing.T) {
		t.Parallel()
		outcome := e.ValidateExpressionDetailed(
			t.Context(),
			"intVar % zeroInt",
			map[string]any{"intVar": 5, "zeroInt": 0},
		)
		require.NoError(t, outcome.Err)
		assert.NotEmpty(t, outcome.Warning)
	})

	t.Run("non numeric result is an error", func(t *testing.T) {
		t.Parallel()
		outcome := e.ValidateExpressionDetailed(t.Context(), `"abc"`, map[string]any{})
		require.Error(t, outcome.Err)
	})

	t.Run("compile failure is an error", func(t *testing.T) {
		t.Parallel()
		outcome := e.ValidateExpressionDetailed(t.Context(), "1 +* 2", map[string]any{})
		require.Error(t, outcome.Err)
	})

	t.Run("valid expression has no error and no warning", func(t *testing.T) {
		t.Parallel()
		outcome := e.ValidateExpressionDetailed(
			t.Context(),
			"round(weight * 2, 2)",
			map[string]any{"weight": 10.0},
		)
		require.NoError(t, outcome.Err)
		assert.Empty(t, outcome.Warning)
	})
}
