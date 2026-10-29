//go:build integration

package driverportalrepository

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A driver's role reads neither workers nor shipments, so the read scope they
// are sent narrows neither kind (readscopeservice sends nothing for a kind the
// person cannot read) and row-level security shows the whole tenant's workers
// and shipments. The driver portal still shows the driver only their own
// worker and the loads they are assigned to drive, because it asks the
// database for exactly those: the worker linked to the caller's user, and
// assignments naming that worker.
func TestDriverPortal_UnderRowLevelSecurityReadsOnlyTheDriversOwnRecords(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	newWorker := func(first string, userID pulid.ID) *worker.Worker {
		entity := &worker.Worker{
			ID:             pulid.MustNew("wrk_"),
			BusinessUnitID: tenant.BuID,
			OrganizationID: tenant.OrgID,
			StateID:        data.State.ID,
			UserID:         userID,
			Status:         domaintypes.StatusActive,
			Type:           worker.WorkerTypeEmployee,
			DriverType:     worker.DriverTypeOTR,
			FirstName:      first,
			LastName:       "Driver",
			AddressLine1:   "1 Depot Road",
			City:           "Springfield",
			PostalCode:     "62701",
			Email: strings.ToLower(
				first,
			) + "-" + pulid.MustNew("e").
				String() +
				"@example.com",
			PhoneNumber:   "555-0100",
			Gender:        worker.GenderMale,
			CanBeAssigned: true,
		}
		require.NoError(t, workerdb.Workers.Create(data.Ctx(ctx), admin, entity))
		return entity
	}
	me := newWorker("Mine", data.User.ID)
	colleague := newWorker("Other", pulid.Nil)

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, admin, data, tenant)
	assign := func(code string, driver *worker.Worker) *testutil.ShipmentGraph {
		graph := testutil.CreateShipmentGraph(
			t,
			ctx,
			admin,
			fixture,
			tenant,
			testutil.ShipmentGraphParams{
				BOL:          "BOL-" + code,
				ProNumber:    "PRO-" + code,
				ShipmentID:   pulid.MustNew("shp_"),
				MoveStatuses: []shipmentvocab.MoveStatus{shipmentvocab.MoveStatusAssigned},
			},
		)
		err := shipmentdb.Assignments.Create(data.Ctx(ctx), admin, &shipment.Assignment{
			ID:              pulid.MustNew("a_"),
			BusinessUnitID:  tenant.BuID,
			OrganizationID:  tenant.OrgID,
			ShipmentMoveID:  graph.Moves[0].ID,
			PrimaryWorkerID: &driver.ID,
			Status:          shipmentvocab.AssignmentStatusNew,
		})
		require.NoError(t, err)
		return graph
	}
	driven := assign("DRIVEN", me)
	theirs := assign("THEIRS", colleague)

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Logger: zap.NewNop()})
	driverCtx := readscope.With(
		dbscope.WithTenant(t.Context(), tenant.DBTenant()),
		func(context.Context) (readscope.Grants, error) { return readscope.Grants{}, nil },
	)

	own, err := repo.GetWorkerByUserID(driverCtx, tenant)
	require.NoError(t, err)
	assert.Equal(t, me.ID, own.ID, "the portal resolves the worker linked to the caller")

	stranger := tenant
	stranger.UserID = pulid.MustNew("usr_")
	_, err = repo.GetWorkerByUserID(
		readscope.With(
			dbscope.WithTenant(t.Context(), stranger.DBTenant()),
			func(context.Context) (readscope.Grants, error) { return readscope.Grants{}, nil },
		),
		stranger,
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "a user linked to no worker resolves none")

	loads, err := repo.ListWorkerLoads(driverCtx, &portalaccessports.ListWorkerLoadsRequest{
		TenantInfo: tenant,
		WorkerID:   own.ID,
	})
	require.NoError(t, err)
	require.Len(t, loads, 1, "only the loads the driver is assigned to")
	require.NotNil(t, loads[0].ShipmentMove)
	require.NotNil(t, loads[0].ShipmentMove.Shipment)
	assert.Equal(t, driven.Shipment.ID, loads[0].ShipmentMove.Shipment.ID)

	assigned, err := repo.WorkerAssignedToShipment(driverCtx, tenant, own.ID, driven.Shipment.ID)
	require.NoError(t, err)
	assert.True(t, assigned)

	assigned, err = repo.WorkerAssignedToShipment(driverCtx, tenant, own.ID, theirs.Shipment.ID)
	require.NoError(t, err)
	assert.False(t, assigned, "a load another driver drives is not the caller's")
}
