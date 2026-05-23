//go:build integration

package dispatchcontrolrepository

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestGetOrCreateParticipatesInOuterTransaction(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	lock := inttestutil.HoldTxLock(
		t,
		conn,
		ports.TxOptions{},
		func(txCtx context.Context, _ bun.Tx) error {
			created, err := repo.GetOrCreate(txCtx, data.Organization.ID, data.BusinessUnit.ID)
			if err != nil {
				return err
			}
			if created == nil {
				return context.Canceled
			}

			_, err = repo.GetByOrgID(txCtx, repositories.GetDispatchControlRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: data.Organization.ID,
					BuID:  data.BusinessUnit.ID,
				},
			})
			return err
		},
	)
	lock.WaitLocked(t)

	visibleOutsideTx, err := repo.GetByOrgID(ctx, repositories.GetDispatchControlRequest{
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

	visibleAfterCommit, err := repo.GetByOrgID(ctx, repositories.GetDispatchControlRequest{
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

	count, err := db.NewSelect().
		TableExpr("dispatch_controls").
		Where("organization_id = ?", data.Organization.ID).
		Where("business_unit_id = ?", data.BusinessUnit.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
