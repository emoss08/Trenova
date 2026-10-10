package kernelports_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
)

type ctxMarker struct{}

func TestAfterCommitRunsImmediatelyOutsideATransaction(t *testing.T) {
	t.Parallel()

	ran := false
	kernelports.AfterCommit(t.Context(), func(context.Context) { ran = true })

	assert.True(t, ran)
}

func TestAfterCommitWaitsForRunAndKeepsOrder(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	order := make([]int, 0, 3)
	for i := range 3 {
		kernelports.AfterCommit(ctx, func(context.Context) { order = append(order, i) })
	}

	assert.Empty(t, order)

	hooks.Run(t.Context())

	assert.Equal(t, []int{0, 1, 2}, order)
}

func TestAfterCommitHooksRunWithTheContextGivenToRun(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	var got any
	kernelports.AfterCommit(ctx, func(runCtx context.Context) { got = runCtx.Value(ctxMarker{}) })

	hooks.Run(context.WithValue(t.Context(), ctxMarker{}, "committed"))

	assert.Equal(t, "committed", got)
}

func TestAfterCommitDroppedWhenNeverRun(t *testing.T) {
	t.Parallel()

	ctx, _ := kernelports.WithAfterCommitHooks(t.Context())
	ran := false
	kernelports.AfterCommit(ctx, func(context.Context) { ran = true })

	assert.False(t, ran)
}

func TestAfterCommitRegisteredWhileRunningRunsAfterTheQueue(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	order := make([]string, 0, 3)
	kernelports.AfterCommit(ctx, func(context.Context) {
		order = append(order, "first")
		kernelports.AfterCommit(ctx, func(context.Context) { order = append(order, "late") })
	})
	kernelports.AfterCommit(ctx, func(context.Context) { order = append(order, "second") })

	hooks.Run(t.Context())

	assert.Equal(t, []string{"first", "second", "late"}, order)
}

func TestAfterCommitAfterRunRunsImmediatelyWithTheCommittedContext(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	hooks.Run(context.WithValue(t.Context(), ctxMarker{}, "committed"))

	var got any
	kernelports.AfterCommit(ctx, func(runCtx context.Context) { got = runCtx.Value(ctxMarker{}) })

	assert.Equal(t, "committed", got)
}

func TestAfterCommitIgnoresNilCallback(t *testing.T) {
	t.Parallel()

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	kernelports.AfterCommit(ctx, nil)
	kernelports.AfterCommit(t.Context(), nil)

	assert.NotPanics(t, func() { hooks.Run(t.Context()) })
}
