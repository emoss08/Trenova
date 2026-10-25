//go:build integration

package dataentrycontrolrepository

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol/dataentrycontroldb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetOrCreateConcurrentCreatesSingleTenantControl(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantCtx := data.Ctx(ctx)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	const calls = 50
	results := make([]string, calls)
	errs := make([]error, calls)

	var wg sync.WaitGroup
	wg.Add(calls)
	for idx := range calls {
		go func() {
			defer wg.Done()

			entity, err := repo.GetOrCreate(tenantCtx, data.Organization.ID, data.BusinessUnit.ID)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = entity.ID.String()
		}()
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	for _, id := range results {
		require.NotEmpty(t, id)
		assert.Equal(t, results[0], id)
	}

	count, err := dataentrycontroldb.DataEntryControls.Query().Count(tenantCtx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetOrCreateInAReadOnlyTransactionWritesNothing(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantCtx := data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	d := dataentrycontroldb.DataEntryControls
	_, err := d.Where(d.OrganizationID.Eq(data.Organization.ID)).Delete(tenantCtx, db)
	require.NoError(t, err)

	err = conn.Transact(
		tenantCtx,
		kernelports.TxOptions{ReadOnly: true},
		func(txCtx context.Context) error {
			entity, getErr := repo.GetOrCreate(txCtx, data.Organization.ID, data.BusinessUnit.ID)
			require.NoError(t, getErr)
			require.NotNil(t, entity)

			_, queryErr := keel.QueryValue[int](txCtx, conn.Keel(txCtx), "SELECT 1")
			return queryErr
		},
	)
	require.NoError(t, err, "the transaction is still usable after the read")

	count, err := d.Query().Count(tenantCtx, db)
	require.NoError(t, err)
	assert.Zero(t, count, "nothing is written from a read-only transaction")
}
