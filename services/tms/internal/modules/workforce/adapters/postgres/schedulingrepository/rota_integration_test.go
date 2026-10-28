//go:build integration

package schedulingrepository_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/schedulingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRotaAssignedDays_CountsLiveAssignmentsPerRosterDay(t *testing.T) {
	ctx, admin, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	driver := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		StateID:        data.State.ID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "Rota",
		LastName:       "Driver",
		AddressLine1:   "1 Depot Road",
		City:           "Springfield",
		PostalCode:     "62701",
		Email:          "rota-" + strings.ToLower(pulid.MustNew("e").String()) + "@example.com",
		PhoneNumber:    "555-0100",
		Gender:         worker.GenderMale,
		CanBeAssigned:  true,
	}
	require.NoError(t, workerdb.Workers.Create(data.Ctx(ctx), admin, driver))

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, admin, data, tenant)
	graph := testutil.CreateShipmentGraph(
		t,
		ctx,
		admin,
		fixture,
		tenant,
		testutil.ShipmentGraphParams{
			BOL:        "BOL-ROTA",
			ProNumber:  "PRO-ROTA",
			ShipmentID: pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{
				shipment.MoveStatusAssigned,
				shipment.MoveStatusAssigned,
				shipment.MoveStatusAssigned,
			},
		},
	)

	const day = int64(1_767_225_600)
	archivedAt := day + 7_200
	for i, created := range []int64{day + 3_600, day + 18_000, day + 3_600} {
		assignment := &shipment.Assignment{
			ID:              pulid.MustNew("a_"),
			BusinessUnitID:  tenant.BuID,
			OrganizationID:  tenant.OrgID,
			ShipmentMoveID:  graph.Moves[i].ID,
			PrimaryWorkerID: &driver.ID,
			Status:          shipment.AssignmentStatusNew,
			CreatedAt:       created,
		}
		if i == 2 {
			assignment.ArchivedAt = &archivedAt
		}
		require.NoError(t, shipmentdb.Assignments.Create(data.Ctx(ctx), admin, assignment))
	}

	repo := schedulingrepository.New(schedulingrepository.Params{
		DB:          postgres.NewTestConnection(admin),
		Logger:      zap.NewNop(),
		Assignments: crossref.RotaAssignments(),
	})
	rows, err := repo.RotaAssignedDays(
		dbscope.WithTenant(ctx, tenant.DBTenant()),
		&workerscheduleports.RotaQuery{
			TenantInfo: tenant,
			WeekStart:  day - 86_400,
			WeekEnd:    day + 6*86_400,
		},
	)
	require.NoError(t, err)

	mine := make([]workerscheduleports.RotaDayRow, 0, 1)
	for _, row := range rows {
		if row.WorkerID == driver.ID {
			mine = append(mine, row)
		}
	}
	assert.Equal(
		t,
		[]workerscheduleports.RotaDayRow{{WorkerID: driver.ID, DayStart: day, Count: 2}},
		mine,
	)
}
