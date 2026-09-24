package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func newAfterCommitTestConnection(t *testing.T) *Connection {
	t.Helper()

	sqldb, err := sql.Open(sqliteDriverName, "file::memory:")
	require.NoError(t, err)
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.ExecContext(t.Context(), "CREATE TABLE after_commit_rows (id INTEGER)")
	require.NoError(t, err)

	return &Connection{db: db}
}

func countAfterCommitRows(ctx context.Context, t *testing.T, conn *Connection) int {
	t.Helper()

	var count int
	require.NoError(t, conn.DBForContext(ctx).
		NewRaw("SELECT COUNT(*) FROM after_commit_rows").
		Scan(ctx, &count))

	return count
}

func TestWithTxRunsAfterCommitCallbacksOnlyOnceCommitted(t *testing.T) {
	t.Parallel()

	conn := newAfterCommitTestConnection(t)
	order := make([]string, 0, 3)
	seen := -1

	err := conn.WithTx(t.Context(), ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO after_commit_rows (id) VALUES (1)"); err != nil {
			return err
		}

		ports.AfterCommit(ctx, func(hookCtx context.Context) {
			order = append(order, "first")
			seen = countAfterCommitRows(hookCtx, t, conn)
		})
		ports.AfterCommit(ctx, func(context.Context) { order = append(order, "second") })
		order = append(order, "body")

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"body", "first", "second"}, order)
	assert.Equal(t, 1, seen)
}

func TestWithTxDropsAfterCommitCallbacksOnRollback(t *testing.T) {
	t.Parallel()

	conn := newAfterCommitTestConnection(t)
	failure := errors.New("boom")
	ran := false

	err := conn.WithTx(t.Context(), ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO after_commit_rows (id) VALUES (1)"); err != nil {
			return err
		}
		ports.AfterCommit(ctx, func(context.Context) { ran = true })

		return failure
	})

	require.ErrorIs(t, err, failure)
	assert.False(t, ran)
	assert.Zero(t, countAfterCommitRows(t.Context(), t, conn))
}

func TestWithTxNestedAfterCommitWaitsForTheOutermostCommit(t *testing.T) {
	t.Parallel()

	conn := newAfterCommitTestConnection(t)
	ranAfterInner := false
	ran := false

	err := conn.WithTx(t.Context(), ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		if err := conn.WithTx(ctx, ports.TxOptions{}, func(inner context.Context, _ bun.Tx) error {
			ports.AfterCommit(inner, func(context.Context) { ran = true })
			return nil
		}); err != nil {
			return err
		}
		ranAfterInner = ran

		return nil
	})
	require.NoError(t, err)

	assert.False(t, ranAfterInner)
	assert.True(t, ran)
}

func TestWithTxNestedAfterCommitDroppedWhenTheOuterRollsBack(t *testing.T) {
	t.Parallel()

	conn := newAfterCommitTestConnection(t)
	failure := errors.New("outer failed")
	ran := false

	err := conn.WithTx(t.Context(), ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		if err := conn.WithTx(ctx, ports.TxOptions{}, func(inner context.Context, _ bun.Tx) error {
			ports.AfterCommit(inner, func(context.Context) { ran = true })
			return nil
		}); err != nil {
			return err
		}

		return failure
	})

	require.ErrorIs(t, err, failure)
	assert.False(t, ran)
}

func TestWithTxAfterCommitCallbackReadsOutsideTheCommittedTransaction(t *testing.T) {
	t.Parallel()

	conn := newAfterCommitTestConnection(t)
	var hookDB bun.IDB

	err := conn.WithTx(t.Context(), ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		ports.AfterCommit(ctx, func(hookCtx context.Context) { hookDB = conn.DBForContext(hookCtx) })
		return nil
	})
	require.NoError(t, err)

	assert.Same(t, conn.db, hookDB)
}
