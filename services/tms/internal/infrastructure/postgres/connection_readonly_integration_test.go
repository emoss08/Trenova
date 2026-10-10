//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestWithTxMarksAReadOnlyTransaction(t *testing.T) {
	_, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := &Connection{db: db}
	ran := false
	readOnly := false
	nestedReadOnly := false

	err := conn.WithTx(t.Context(), kernelports.TxOptions{ReadOnly: true}, func(ctx context.Context, _ bun.Tx) error {
		readOnly = kernelports.IsReadOnly(ctx)
		kernelports.AfterCommit(ctx, func(context.Context) { ran = true })

		return nil
	})
	require.NoError(t, err)

	err = conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		assert.False(t, kernelports.IsReadOnly(ctx))

		return conn.WithTx(ctx, kernelports.TxOptions{ReadOnly: true}, func(inner context.Context, _ bun.Tx) error {
			nestedReadOnly = kernelports.IsReadOnly(inner)
			return nil
		})
	})
	require.NoError(t, err)

	assert.True(t, readOnly)
	assert.True(t, nestedReadOnly)
	assert.False(t, ran)
}
