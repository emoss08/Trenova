package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	afterCommitInsertSQL = "INSERT INTO after_commit_rows (id) VALUES (1)"
	afterCommitCountSQL  = "SELECT COUNT(*) FROM after_commit_rows"
)

func newAfterCommitTestConnection(t *testing.T) (*Connection, sqlmock.Sqlmock) {
	t.Helper()

	sqldb, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	db := bun.NewDB(sqldb, pgdialect.New())
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})

	return &Connection{db: db}, mock
}

func countAfterCommitRows(ctx context.Context, t *testing.T, conn *Connection) int {
	t.Helper()

	var count int
	require.NoError(t, conn.DBForContext(ctx).
		NewRaw(afterCommitCountSQL).
		Scan(ctx, &count))

	return count
}

func TestWithTxRunsAfterCommitCallbacksOnlyOnceCommitted(t *testing.T) {
	t.Parallel()

	conn, mock := newAfterCommitTestConnection(t)
	mock.ExpectBegin()
	mock.ExpectExec(afterCommitInsertSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(afterCommitCountSQL).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	order := make([]string, 0, 3)
	seen := -1

	err := conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, afterCommitInsertSQL); err != nil {
			return err
		}

		kernelports.AfterCommit(ctx, func(hookCtx context.Context) {
			order = append(order, "first")
			seen = countAfterCommitRows(hookCtx, t, conn)
		})
		kernelports.AfterCommit(ctx, func(context.Context) { order = append(order, "second") })
		order = append(order, "body")

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"body", "first", "second"}, order)
	assert.Equal(t, 1, seen)
}

func TestWithTxDropsAfterCommitCallbacksOnRollback(t *testing.T) {
	t.Parallel()

	conn, mock := newAfterCommitTestConnection(t)
	mock.ExpectBegin()
	mock.ExpectExec(afterCommitInsertSQL).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	failure := errors.New("boom")
	ran := false

	err := conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, afterCommitInsertSQL); err != nil {
			return err
		}
		kernelports.AfterCommit(ctx, func(context.Context) { ran = true })

		return failure
	})

	require.ErrorIs(t, err, failure)
	assert.False(t, ran)
}

func TestWithTxNestedAfterCommitWaitsForTheOutermostCommit(t *testing.T) {
	t.Parallel()

	conn, mock := newAfterCommitTestConnection(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	ranAfterInner := false
	ran := false

	err := conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		if err := conn.WithTx(ctx, kernelports.TxOptions{}, func(inner context.Context, _ bun.Tx) error {
			kernelports.AfterCommit(inner, func(context.Context) { ran = true })
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

	conn, mock := newAfterCommitTestConnection(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	failure := errors.New("outer failed")
	ran := false

	err := conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		if err := conn.WithTx(ctx, kernelports.TxOptions{}, func(inner context.Context, _ bun.Tx) error {
			kernelports.AfterCommit(inner, func(context.Context) { ran = true })
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

	conn, mock := newAfterCommitTestConnection(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var hookDB bun.IDB

	err := conn.WithTx(t.Context(), kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		kernelports.AfterCommit(ctx, func(hookCtx context.Context) { hookDB = conn.DBForContext(hookCtx) })
		return nil
	})
	require.NoError(t, err)

	assert.Same(t, conn.db, hookDB)
}
