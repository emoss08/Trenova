//go:build integration

package dispatchconsolerepository_test

import (
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity/equipmentcontinuitydb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchconsolerepository"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
The inspector ranked a move's candidates from the first 500 dispatchable drivers
by name while the plan weighed every driver, so the plan's pick for a move was
often missing from the inspector's list. A driver read near a point now reads
the drivers nearest it first, each placed where the candidate ranking starts
them: where the load they are on ends, else a fresh position, else the
tractor's last drop.
*/
func TestListBoardDrivers_ReadsTheDriversNearestThePointFirst(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	trc := tractordb.Tractors
	owned, err := trc.AllTenants().
		Where(trc.PrimaryWorkerID.IsNotNull()).
		Order(trc.ID.Asc()).
		First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: owned.OrganizationID, BuID: owned.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	repo := dispatchconsolerepository.New(dispatchconsolerepository.Params{
		DB:     postgres.WrapDB(db),
		Logger: zap.NewNop(),
	})
	all, err := repo.ListBoardDrivers(ctx, &dispatchboardports.DispatchBoardFilter{
		TenantInfo:  tenant,
		DriverLimit: 10_000,
	})
	require.NoError(t, err)
	var pair []*dispatchboardports.BoardDriver
	var busy *dispatchboardports.BoardDriver
	for _, driver := range all.Drivers {
		if driver.TractorID.IsNil() {
			continue
		}
		if driver.OpenAssignments == 1 && busy == nil {
			busy = driver
			continue
		}
		if driver.OpenAssignments > 0 || len(pair) == 2 {
			continue
		}
		if len(pair) == 1 && driver.AvailableForDispatch != pair[0].AvailableForDispatch {
			continue
		}
		pair = append(pair, driver)
	}
	require.Len(t, pair, 2, "the seed has two idle drivers with tractors")

	first := geoutils.Point{Latitude: 47.123456, Longitude: -122.654321}
	second := geoutils.Point{Latitude: 25.111111, Longitude: -80.222222}
	now := time.Now().Unix()
	vp := telematicsdb.VehiclePositions
	upsert := func(positions ...*telematics.VehiclePosition) {
		_, upsertErr := vp.Upsert(ctx, db, positions,
			keel.OnConflict(vp.OrganizationID, vp.BusinessUnitID, vp.TractorID).
				Update(vp.Latitude, vp.Longitude, vp.RecordedAt, vp.ReceivedAt))
		require.NoError(t, upsertErr)
	}
	upsert(
		position(tenant, pair[0].TractorID, first, now),
		position(tenant, pair[1].TractorID, second, now),
	)

	nearest := func(point geoutils.Point, workers ...pulid.ID) pulid.ID {
		page, readErr := repo.ListBoardDrivers(ctx, &dispatchboardports.DispatchBoardFilter{
			TenantInfo:  tenant,
			WorkerIDs:   workers,
			DriverLimit: 1,
			Near:        &point,
		})
		require.NoError(t, readErr)
		require.Len(t, page.Drivers, 1)
		assert.True(t, page.Truncated)
		return page.Drivers[0].WorkerID
	}

	assert.Equal(t, pair[0].WorkerID, nearest(first, pair[0].WorkerID, pair[1].WorkerID))
	assert.Equal(t, pair[1].WorkerID, nearest(second, pair[0].WorkerID, pair[1].WorkerID))

	t.Run("a driver on a load is placed where the load ends", func(t *testing.T) {
		if busy == nil || busy.AvailableForDispatch != pair[0].AvailableForDispatch {
			t.Skip("the seed has no driver on exactly one load beside the idle pair")
		}
		commitments, err := repo.ListWorkerCommitments(
			ctx,
			&dispatchboardports.ListWorkerWindowsRequest{
				TenantInfo: tenant,
				WorkerIDs:  []pulid.ID{busy.WorkerID},
			},
		)
		require.NoError(t, err)
		require.Len(t, commitments, 1)
		require.NotNil(t, commitments[0].DestLatitude)
		require.NotNil(t, commitments[0].DestLongitude)
		ends := geoutils.Point{
			Latitude:  *commitments[0].DestLatitude,
			Longitude: *commitments[0].DestLongitude,
		}
		upsert(position(tenant, busy.TractorID, first, now))

		assert.Equal(t, busy.WorkerID, nearest(ends, pair[0].WorkerID, busy.WorkerID))
		assert.Equal(
			t,
			pair[0].WorkerID,
			nearest(first, pair[0].WorkerID, busy.WorkerID),
			"where the truck is now does not count while it is on a load",
		)
	})

	_, err = vp.Where(vp.TractorID.Eq(pair[1].TractorID)).
		Update(ctx, db, vp.RecordedAt.To(now-5*3600))
	require.NoError(t, err)
	ec := equipmentcontinuitydb.EquipmentContinuities
	_, err = ec.Where(ec.EquipmentID.Eq(pair[1].TractorID), ec.IsCurrent.IsTrue()).
		Update(ctx, db, ec.IsCurrent.To(false))
	require.NoError(t, err)

	assert.Equal(
		t,
		pair[0].WorkerID,
		nearest(second, pair[0].WorkerID, pair[1].WorkerID),
		"a stale position is no position, and a driver with no position comes last",
	)
}

func position(
	tenant pagination.TenantInfo,
	tractorID pulid.ID,
	at geoutils.Point,
	recordedAt int64,
) *telematics.VehiclePosition {
	return &telematics.VehiclePosition{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		TractorID:         tractorID,
		Provider:          "samsara",
		ProviderVehicleID: "veh-" + tractorID.String(),
		Latitude:          at.Latitude,
		Longitude:         at.Longitude,
		RecordedAt:        recordedAt,
		ReceivedAt:        recordedAt,
	}
}
