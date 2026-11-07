//go:build integration

package driverpositioningrepository_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/driverpositioningrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	ctx      context.Context
	repo     driverpositioningports.Repository
	tenant   pagination.TenantInfo
	workerID pulid.ID
	now      int64
}

func setup(t *testing.T) (fixture, func(at int64)) {
	t.Helper()
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	a := shipmentdb.Assignments
	assignment, err := a.AllTenants().
		Where(a.PrimaryWorkerID.IsNotNull(), a.ArchivedAt.IsNull()).
		Order(a.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{
		OrgID: assignment.OrganizationID,
		BuID:  assignment.BusinessUnitID,
	}
	tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())

	reassign := func(at int64) {
		_, err := a.Where(a.ID.Eq(assignment.ID)).Update(tenantCtx, db, a.CreatedAt.To(at))
		require.NoError(t, err)
	}

	return fixture{
		ctx: tenantCtx,
		repo: driverpositioningrepository.New(driverpositioningrepository.Params{
			DB:     postgres.WrapDB(db),
			Logger: zap.NewNop(),
		}),
		tenant:   tenant,
		workerID: *assignment.PrimaryWorkerID,
		now:      time.Now().Unix(),
	}, reassign
}

func positioningFor(f fixture) *driverpositioning.Positioning {
	slot := dispatchmarket.SlotStart(f.now) + dispatchmarket.SlotSeconds
	return &driverpositioning.Positioning{
		WorkerID:    f.workerID,
		ToCell:      12971,
		ToLabel:     "Chicago, IL",
		ToLatitude:  41.88,
		ToLongitude: -87.63,
		SlotStart:   slot,
		FreeAt:      f.now,
		ArriveAt:    f.now + 3600,
		Status:      driverpositioning.StatusActive,
	}
}

func TestReplace_KeepsOneActivePositioningPerDriver(t *testing.T) {
	f, _ := setup(t)
	ctx := f.ctx

	first, err := f.repo.Replace(ctx, &driverpositioningports.ReplaceRequest{
		TenantInfo: f.tenant, Positioning: positioningFor(f),
	})
	require.NoError(t, err)
	assert.Nil(t, first.Previous)
	assert.False(t, first.Positioning.ID.IsNil())

	second, err := f.repo.Replace(ctx, &driverpositioningports.ReplaceRequest{
		TenantInfo: f.tenant, Positioning: positioningFor(f),
	})
	require.NoError(t, err)
	require.NotNil(t, second.Previous)
	assert.Equal(t, first.Positioning.ID, second.Previous.ID)
	assert.Equal(t, driverpositioning.StatusReplaced, second.Previous.Status)

	stored, err := f.repo.Get(ctx, &driverpositioningports.GetRequest{
		TenantInfo: f.tenant, ID: first.Positioning.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, driverpositioning.StatusReplaced, stored.Status)
}

func TestCancel_CallsAnActivePositioningOffOnce(t *testing.T) {
	f, _ := setup(t)
	ctx := f.ctx
	created, err := f.repo.Replace(ctx, &driverpositioningports.ReplaceRequest{
		TenantInfo: f.tenant, Positioning: positioningFor(f),
	})
	require.NoError(t, err)

	canceled, err := f.repo.Cancel(ctx, &driverpositioningports.CancelRequest{
		TenantInfo: f.tenant, ID: created.Positioning.ID, CanceledByID: "usr_x", CanceledAt: f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, driverpositioning.StatusCanceled, canceled.Status)
	require.NotNil(t, canceled.CanceledAt)
	assert.Equal(t, f.now, *canceled.CanceledAt)

	_, err = f.repo.Cancel(ctx, &driverpositioningports.CancelRequest{
		TenantInfo: f.tenant, ID: created.Positioning.ID, CanceledAt: f.now,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business, "a positioning already off cannot be called off again")

	_, err = f.repo.Cancel(ctx, &driverpositioningports.CancelRequest{
		TenantInfo: f.tenant, ID: pulid.MustNew("dpo_"), CanceledAt: f.now,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestListLive_DropsCalledOffExpiredAndSupersededPositionings(t *testing.T) {
	f, reassign := setup(t)
	ctx := f.ctx
	created, err := f.repo.Replace(ctx, &driverpositioningports.ReplaceRequest{
		TenantInfo: f.tenant, Positioning: positioningFor(f),
	})
	require.NoError(t, err)
	reassign(created.Positioning.CreatedAt - 60)

	live := func(now int64) []*driverpositioning.Positioning {
		rows, listErr := f.repo.ListLive(ctx, &driverpositioningports.LiveRequest{
			TenantInfo: f.tenant, WorkerIDs: []pulid.ID{f.workerID}, Now: now,
		})
		require.NoError(t, listErr)
		return rows
	}

	require.Len(t, live(f.now), 1, "a positioning newer than the driver's last assignment is live")
	assert.Empty(t, live(created.Positioning.ExpiresAt()+dispatchmarket.SlotSeconds),
		"it expires twelve hours after its window ends")

	reassign(created.Positioning.CreatedAt + 60)
	assert.Empty(t, live(f.now), "an assignment made after it supersedes it")

	reassign(created.Positioning.CreatedAt - 60)
	_, err = f.repo.Cancel(ctx, &driverpositioningports.CancelRequest{
		TenantInfo: f.tenant, ID: created.Positioning.ID, CanceledAt: f.now,
	})
	require.NoError(t, err)
	assert.Empty(t, live(f.now), "a called-off positioning is not live")

}
