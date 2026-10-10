package kernelports_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
)

func TestIsReadOnly(t *testing.T) {
	t.Parallel()

	assert.False(t, kernelports.IsReadOnly(t.Context()))
	assert.True(t, kernelports.IsReadOnly(kernelports.WithReadOnly(t.Context())))
}

func TestAfterCommitDropsCallbacksQueuedInAReadOnlyContext(t *testing.T) {
	t.Parallel()

	ran := false
	ctx, hooks := kernelports.WithAfterCommitHooks(kernelports.WithReadOnly(t.Context()))
	kernelports.AfterCommit(ctx, func(context.Context) { ran = true })
	hooks.Run(t.Context())

	assert.False(t, ran)
}

func TestAfterCommitDropsCallbacksOutsideATransactionWhenReadOnly(t *testing.T) {
	t.Parallel()

	ran := false
	kernelports.AfterCommit(kernelports.WithReadOnly(t.Context()), func(context.Context) { ran = true })

	assert.False(t, ran)
}
