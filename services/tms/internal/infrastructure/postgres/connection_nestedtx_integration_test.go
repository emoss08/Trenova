//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestNestedRunInTxStaysInsideTheOuterTransaction(t *testing.T) {
	_, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx := t.Context()
	_, err := db.ExecContext(ctx, "CREATE TABLE nested_tx_probe (id INT PRIMARY KEY)")
	require.NoError(t, err)

	conn := &Connection{db: db}
	rollback := errors.New("roll back the outer transaction")
	hookRan := false

	err = conn.WithTx(ctx, kernelports.TxOptions{}, func(outer context.Context, _ bun.Tx) error {
		kernelports.AfterCommit(outer, func(context.Context) { hookRan = true })

		nestedErr := conn.DBForContext(outer).RunInTx(
			outer,
			nil,
			func(inner context.Context, _ bun.Tx) error {
				_, isTx := conn.DBForContext(inner).(bun.Tx)
				assert.True(t, isTx, "a nested callback must still see the outer transaction")

				_, insertErr := conn.DBForContext(inner).
					NewRaw("INSERT INTO nested_tx_probe (id) VALUES (1)").
					Exec(inner)
				return insertErr
			},
		)
		if nestedErr != nil {
			return nestedErr
		}

		return rollback
	})
	require.ErrorIs(t, err, rollback)

	var count int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM nested_tx_probe").Scan(ctx, &count))
	assert.Equal(t, 0, count, "a write in a nested transaction must roll back with the outer one")
	assert.False(t, hookRan)
}
