//go:build integration

package dispatchcontrolrepository

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetOrCreateParticipatesInOuterTransaction(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	lock := inttestutil.HoldTxLock(
		t,
		ctx,
		conn,
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			created, err := repo.GetOrCreate(txCtx, data.Organization.ID, data.BusinessUnit.ID)
			if err != nil {
				return err
			}
			if created == nil {
				return context.Canceled
			}

			_, err = repo.GetByOrgID(txCtx, dispatchcontrolports.GetDispatchControlRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: data.Organization.ID,
					BuID:  data.BusinessUnit.ID,
				},
			})
			return err
		},
	)
	lock.WaitLocked(t)

	visibleOutsideTx, err := repo.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
	})
	require.Nil(t, visibleOutsideTx)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	lock.Release()
	lock.Wait(t)

	visibleAfterCommit, err := repo.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, visibleAfterCommit)
}

func TestGetOrCreateConcurrentCreatesSingleTenantControl(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
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

			entity, err := repo.GetOrCreate(ctx, data.Organization.ID, data.BusinessUnit.ID)
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

	count, err := dispatchcontroldb.DispatchControls.Query().Count(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestListHorizonPlanningTenantsReturnsHorizonTenants(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	tenant := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	before, err := repo.ListHorizonPlanningTenants(ctx)
	require.NoError(t, err)
	assert.NotContains(t, before, tenant)

	control, err := repo.GetOrCreate(ctx, data.Organization.ID, data.BusinessUnit.ID)
	require.NoError(t, err)

	control.EnableAutoAssignment = true
	control.PlanningMode = dispatchcontrol.PlanningModeHorizon
	_, err = repo.Update(ctx, control)
	require.NoError(t, err)

	after, err := repo.ListHorizonPlanningTenants(ctx)
	require.NoError(t, err)
	assert.Contains(t, after, tenant)
}

// Assigning a driver plans against the dispatch control, and a proposal's
// preview runs that plan in a read-only snapshot. Creating the missing row
// there aborted the transaction, so every check after it failed.
func TestGetOrCreateInAReadOnlyTransactionWritesNothing(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	dc := dispatchcontroldb.DispatchControls
	_, err := dc.Where(keel.True()).Delete(ctx, db)
	require.NoError(t, err)

	err = conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: true},
		func(txCtx context.Context) error {
			entity, getErr := repo.GetOrCreate(txCtx, data.Organization.ID, data.BusinessUnit.ID)
			require.NoError(t, getErr)
			require.NotNil(t, entity)

			_, err := keel.QueryValue[int](txCtx, conn.Keel(txCtx), "SELECT 1")
			return err
		},
	)
	require.NoError(t, err, "the transaction is still usable after the read")

	count, err := dc.Query().Count(ctx, db)
	require.NoError(t, err)
	assert.Zero(t, count, "nothing is written from a read-only transaction")
}
