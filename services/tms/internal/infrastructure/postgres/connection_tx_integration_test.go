//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTxTestConnection(t *testing.T) (*Connection, *keel.DB) {
	t.Helper()

	_, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	_, err := keel.Exec(t.Context(), db, "CREATE TABLE tx_probe (id INT PRIMARY KEY)")
	require.NoError(t, err)

	return WrapDB(db), db
}

func countProbeRows(ctx context.Context, t *testing.T, db *keel.DB) int64 {
	t.Helper()

	n, err := keel.QueryValue[int64](ctx, db, "SELECT count(*) FROM tx_probe")
	require.NoError(t, err)

	return n
}

func insertProbe(ctx context.Context, db *keel.DB, id int) error {
	_, err := keel.Exec(ctx, db, "INSERT INTO tx_probe (id) VALUES ($1)", id)
	return err
}

func TestTransactRunsAfterCommitCallbacksOnceCommitted(t *testing.T) {
	conn, db := newTxTestConnection(t)

	order := make([]string, 0, 3)
	seen := int64(-1)

	err := conn.Transact(t.Context(), kernelports.TxOptions{}, func(ctx context.Context) error {
		if err := insertProbe(ctx, db, 1); err != nil {
			return err
		}

		kernelports.AfterCommit(ctx, func(hookCtx context.Context) {
			order = append(order, "first")
			seen = countProbeRows(hookCtx, t, db)
		})
		kernelports.AfterCommit(ctx, func(context.Context) { order = append(order, "second") })
		order = append(order, "body")

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"body", "first", "second"}, order)
	assert.Equal(t, int64(1), seen)
}

func TestTransactRollsBackAndDropsCallbacksOnError(t *testing.T) {
	conn, db := newTxTestConnection(t)

	failure := errors.New("boom")
	ran := false

	err := conn.Transact(t.Context(), kernelports.TxOptions{}, func(ctx context.Context) error {
		if err := insertProbe(ctx, db, 1); err != nil {
			return err
		}
		kernelports.AfterCommit(ctx, func(context.Context) { ran = true })

		return failure
	})

	require.ErrorIs(t, err, failure)
	assert.False(t, ran)
	assert.Equal(t, int64(0), countProbeRows(t.Context(), t, db))
}

func TestNestedTransactJoinsTheOuterTransaction(t *testing.T) {
	conn, db := newTxTestConnection(t)

	rollback := errors.New("roll back the outer transaction")
	ranAfterInner := false
	ran := false

	err := conn.Transact(t.Context(), kernelports.TxOptions{}, func(outer context.Context) error {
		if err := conn.Transact(outer, kernelports.TxOptions{}, func(inner context.Context) error {
			kernelports.AfterCommit(inner, func(context.Context) { ran = true })
			return insertProbe(inner, db, 1)
		}); err != nil {
			return err
		}
		ranAfterInner = ran

		return rollback
	})

	require.ErrorIs(t, err, rollback)
	assert.False(t, ranAfterInner)
	assert.False(t, ran)
	assert.Equal(t, int64(0), countProbeRows(t.Context(), t, db),
		"a write in a nested transaction must roll back with the outer one")
}

func TestSavepointRollsBackOnlyItsOwnWork(t *testing.T) {
	conn, db := newTxTestConnection(t)

	failure := errors.New("undo the savepoint")

	err := conn.Transact(t.Context(), kernelports.TxOptions{}, func(ctx context.Context) error {
		if err := insertProbe(ctx, db, 1); err != nil {
			return err
		}

		spErr := conn.Savepoint(ctx, func(ctx context.Context) error {
			if err := insertProbe(ctx, db, 2); err != nil {
				return err
			}
			return failure
		})
		require.ErrorIs(t, spErr, failure)

		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), countProbeRows(t.Context(), t, db))
}

func TestTransactMarksAReadOnlyTransaction(t *testing.T) {
	conn, db := newTxTestConnection(t)

	ran := false
	readOnly := false
	nestedReadOnly := false

	err := conn.Transact(t.Context(), kernelports.TxOptions{ReadOnly: true}, func(ctx context.Context) error {
		readOnly = kernelports.IsReadOnly(ctx)
		kernelports.AfterCommit(ctx, func(context.Context) { ran = true })

		return nil
	})
	require.NoError(t, err)

	err = conn.Transact(t.Context(), kernelports.TxOptions{ReadOnly: true}, func(ctx context.Context) error {
		return insertProbe(ctx, db, 1)
	})
	require.Error(t, err, "a read-only transaction refuses writes")

	err = conn.Transact(t.Context(), kernelports.TxOptions{}, func(ctx context.Context) error {
		assert.False(t, kernelports.IsReadOnly(ctx))

		return conn.Transact(ctx, kernelports.TxOptions{ReadOnly: true}, func(inner context.Context) error {
			nestedReadOnly = kernelports.IsReadOnly(inner)
			return nil
		})
	})
	require.NoError(t, err)

	assert.True(t, readOnly)
	assert.True(t, nestedReadOnly)
	assert.False(t, ran)
}

func TestTransactUnderAnotherScopeOpensItsOwnTransaction(t *testing.T) {
	conn, db := newTxTestConnection(t)

	tenant := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	})
	rollback := errors.New("roll back the tenant transaction")

	err := conn.Transact(tenant, kernelports.TxOptions{}, func(ctx context.Context) error {
		system := dbscope.WithSystem(ctx, "tx probe")
		if err := conn.Transact(system, kernelports.TxOptions{}, func(ctx context.Context) error {
			return insertProbe(ctx, db, 1)
		}); err != nil {
			return err
		}

		return rollback
	})

	require.ErrorIs(t, err, rollback)
	assert.Equal(t, int64(1), countProbeRows(t.Context(), t, db),
		"work under another scope commits in its own transaction")
}

func TestRunDetachedCommitsWhateverTheCallerDoes(t *testing.T) {
	conn, db := newTxTestConnection(t)
	conn.rls = &rlsRuntime{}

	rollback := errors.New("roll back the outer transaction")

	err := conn.Transact(t.Context(), kernelports.TxOptions{}, func(ctx context.Context) error {
		if err := conn.RunDetached(ctx, false, func(ctx context.Context) error {
			return insertProbe(ctx, db, 1)
		}); err != nil {
			return err
		}

		return rollback
	})

	require.ErrorIs(t, err, rollback)
	assert.Equal(t, int64(1), countProbeRows(t.Context(), t, db))
}

func TestTransactAppliesIsolationAndLockTimeout(t *testing.T) {
	conn, db := newTxTestConnection(t)

	err := conn.Transact(t.Context(), kernelports.TxOptions{
		Isolation:   sql.LevelRepeatableRead,
		LockTimeout: 1500 * time.Millisecond,
	}, func(ctx context.Context) error {
		isolation, err := keel.QueryValue[string](ctx, db, "SHOW transaction_isolation")
		if err != nil {
			return err
		}
		assert.Equal(t, "repeatable read", isolation)

		timeout, err := keel.QueryValue[string](ctx, db, "SHOW lock_timeout")
		if err != nil {
			return err
		}
		assert.Equal(t, "1500ms", timeout)

		return nil
	})
	require.NoError(t, err)

	err = conn.Transact(t.Context(), kernelports.TxOptions{Isolation: sql.LevelSnapshot}, func(context.Context) error {
		return nil
	})
	require.ErrorIs(t, err, ErrUnsupportedIsolation)
}
