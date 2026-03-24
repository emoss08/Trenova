//go:build integration

package dispatchcontrolrepository

import (
	"context"
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
