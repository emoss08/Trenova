package ports_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/assert"
)

type ctxMarker struct{}

func TestAfterCommitRunsImmediatelyOutsideATransaction(t *testing.T) {
	t.Parallel()

	ran := false
	ports.AfterCommit(t.Context(), func(context.Context) { ran = true })

	assert.True(t, ran)
}

func TestAfterCommitWaitsForRunAndKeepsOrder(t *testing.T) {
	t.Parallel()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	order := make([]int, 0, 3)
	for i := range 3 {
		ports.AfterCommit(ctx, func(context.Context) { order = append(order, i) })
	}

	assert.Empty(t, order)

	hooks.Run(t.Context())

	assert.Equal(t, []int{0, 1, 2}, order)
}

func TestAfterCommitHooksRunWithTheContextGivenToRun(t *testing.T) {
	t.Parallel()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	var got any
	ports.AfterCommit(ctx, func(runCtx context.Context) { got = runCtx.Value(ctxMarker{}) })

	hooks.Run(context.WithValue(t.Context(), ctxMarker{}, "committed"))

	assert.Equal(t, "committed", got)
}

func TestAfterCommitDroppedWhenNeverRun(t *testing.T) {
	t.Parallel()

	ctx, _ := ports.WithAfterCommitHooks(t.Context())
	ran := false
	ports.AfterCommit(ctx, func(context.Context) { ran = true })

	assert.False(t, ran)
}

func TestAfterCommitRegisteredWhileRunningRunsAfterTheQueue(t *testing.T) {
	t.Parallel()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	order := make([]string, 0, 3)
	ports.AfterCommit(ctx, func(context.Context) {
		order = append(order, "first")
		ports.AfterCommit(ctx, func(context.Context) { order = append(order, "late") })
	})
	ports.AfterCommit(ctx, func(context.Context) { order = append(order, "second") })

	hooks.Run(t.Context())

	assert.Equal(t, []string{"first", "second", "late"}, order)
}

func TestAfterCommitAfterRunRunsImmediatelyWithTheCommittedContext(t *testing.T) {
	t.Parallel()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	hooks.Run(context.WithValue(t.Context(), ctxMarker{}, "committed"))

	var got any
	ports.AfterCommit(ctx, func(runCtx context.Context) { got = runCtx.Value(ctxMarker{}) })

	assert.Equal(t, "committed", got)
}

func TestAfterCommitIgnoresNilCallback(t *testing.T) {
	t.Parallel()

	ctx, hooks := ports.WithAfterCommitHooks(t.Context())
	ports.AfterCommit(ctx, nil)
	ports.AfterCommit(t.Context(), nil)

	assert.NotPanics(t, func() { hooks.Run(t.Context()) })
}
