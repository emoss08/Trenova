package ports_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/assert"
)

func TestIsReadOnly(t *testing.T) {
	t.Parallel()

	assert.False(t, ports.IsReadOnly(t.Context()))
	assert.True(t, ports.IsReadOnly(ports.WithReadOnly(t.Context())))
}

func TestAfterCommitDropsCallbacksQueuedInAReadOnlyContext(t *testing.T) {
	t.Parallel()

	ran := false
	ctx, hooks := ports.WithAfterCommitHooks(ports.WithReadOnly(t.Context()))
	ports.AfterCommit(ctx, func(context.Context) { ran = true })
	hooks.Run(t.Context())

	assert.False(t, ran)
}

func TestAfterCommitDropsCallbacksOutsideATransactionWhenReadOnly(t *testing.T) {
	t.Parallel()

	ran := false
	ports.AfterCommit(ports.WithReadOnly(t.Context()), func(context.Context) { ran = true })

	assert.False(t, ran)
}
