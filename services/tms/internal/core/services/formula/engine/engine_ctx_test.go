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

// slowFn blocks the VM inside a function call for the given time. Cancellation
// checks run between VM instructions, so this is the shape of evaluation that
// outlives the caller's deadline: the caller returns while the goroutine is
// still inside the call.
func slowFn(d time.Duration) func() float64 {
	return func() float64 {
		time.Sleep(d)
		return 1
	}
}

// TestEvaluateWithEnv_SlowExpressionIsCancelled guards the deadline: the
// caller gets its answer at the deadline, not when the evaluation finally
// finishes, and the goroutine still exits on its own afterwards.
func TestEvaluateWithEnv_SlowExpressionIsCancelled(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)

	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()

	before := runtime.NumGoroutine()
	start := time.Now()

	_, err := e.EvaluateWithEnv(
		ctx,
		&formulatemplatetypes.EnvEvaluationRequest{
			Expression: "slow() * 2",
			Env:        map[string]any{"slow": slowFn(time.Second)},
		},
	)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 900*time.Millisecond, "evaluation must stop at cancellation")

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

// TestRun_TimeoutLeavesCallerEnvUntouched guards the environment copy: after a
// timed-out evaluation the caller's map must be exactly as it was, with no
// injected context key and every entry intact, and must be safe to reuse.
func TestRun_TimeoutLeavesCallerEnvUntouched(t *testing.T) {
	t.Parallel()

	e := newBareEngine(t)
	ctx := engine.WithEvaluationTimeout(t.Context(), 100*time.Millisecond)

	env := map[string]any{"weight": 10.0, "rate": 2.5, "slow": slowFn(400 * time.Millisecond)}

	start := time.Now()
	_, err := e.EvaluateWithEnv(ctx, &formulatemplatetypes.EnvEvaluationRequest{
		Expression: "slow() * weight",
		Env:        env,
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 350*time.Millisecond, "the override must shorten the leash")

	// The abandoned goroutine is still inside slow() here; the caller's map
	// must already be clean and must stay clean once that goroutine finishes.
	assert.NotContains(t, env, "__ctx")
	assert.InDelta(t, 10.0, env["weight"], 0)
	time.Sleep(500 * time.Millisecond)
	assert.NotContains(t, env, "__ctx")
	assert.InDelta(t, 2.5, env["rate"], 0)

	result, err := e.EvaluateWithEnv(t.Context(), &formulatemplatetypes.EnvEvaluationRequest{
		Expression: "weight * rate",
		Env:        env,
	})
	require.NoError(t, err)
	assert.Equal(t, "25", result.Value.String())
	assert.NotContains(t, env, "__ctx")
}

func TestWithEvaluationTimeout_IgnoresNonPositive(t *testing.T) {
	t.Parallel()

	base := t.Context()
	assert.Equal(t, base, engine.WithEvaluationTimeout(base, 0))
	assert.Equal(t, base, engine.WithEvaluationTimeout(base, -time.Second))
	assert.NotEqual(t, base, engine.WithEvaluationTimeout(base, time.Second))
}
