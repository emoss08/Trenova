//go:build integration

package servicefailurerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurerepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func recordFailureAt(
	ctx context.Context,
	t *testing.T,
	db *keel.DB,
	stop *shipment.Stop,
) pulid.ID {
	t.Helper()

	move, err := shipmentdb.ShipmentMoves.Get(ctx, db, stop.ShipmentMoveID)
	require.NoError(t, err)

	id := pulid.MustNew("sf_")
	failure := &servicefailure.ServiceFailure{
		ID:              id,
		OrganizationID:  stop.OrganizationID,
		BusinessUnitID:  stop.BusinessUnitID,
		ShipmentID:      move.ShipmentID,
		ShipmentMoveID:  stop.ShipmentMoveID,
		StopID:          stop.ID,
		Number:          "SF-" + id.String()[len(id.String())-8:],
		Type:            servicefailure.TypeLateDelivery,
		StopType:        stop.Type,
		ScheduledCutoff: 1000,
		ActualArrival:   4600,
		LateMinutes:     60,
	}
	require.NoError(t, servicefailuredb.ServiceFailures.Create(ctx, db, failure))

	return id
}

func atDistinctLocations(stops []*shipment.Stop, n int) []*shipment.Stop {
	seen := make(map[pulid.ID]bool, n)
	picked := make([]*shipment.Stop, 0, n)
	for _, stop := range stops {
		if seen[stop.LocationID] {
			continue
		}
		seen[stop.LocationID] = true
		picked = append(picked, stop)
		if len(picked) == n {
			break
		}
	}

	return picked
}

func TestListConnection_NarrowsToFailuresAtTheLocationAsked(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	st := shipmentdb.Stops
	first, err := st.AllTenants().Order(st.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: first.OrganizationID,
		BusinessUnitID: first.BusinessUnitID,
	})

	candidates, err := st.Order(st.ID.Asc()).Limit(200).List(ctx, db)
	require.NoError(t, err)
	stops := atDistinctLocations(candidates, 2)
	require.Len(t, stops, 2, "the development seed should hold stops at two locations")

	here := recordFailureAt(ctx, t, db, stops[0])
	there := recordFailureAt(ctx, t, db, stops[1])

	repo := servicefailurerepository.New(servicefailurerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: stops[0].OrganizationID, BuID: stops[0].BusinessUnitID}
	at := stops[0].LocationID
	result, err := repo.ListConnection(
		ctx,
		&servicefailureports.ListServiceFailureConnectionRequest{
			Filter:     &pagination.QueryOptions{TenantInfo: tenant},
			Cursor:     pagination.CursorInfo{Limit: 100, IncludeTotalCount: true},
			LocationID: &at,
		},
	)
	require.NoError(t, err)

	found := make(map[pulid.ID]bool, len(result.Items))
	for _, failure := range result.Items {
		found[failure.ID] = true
		require.NotNil(t, failure.Stop)
		assert.Equal(t, at, failure.Stop.LocationID, "%s is at another location", failure.Number)
	}
	assert.True(t, found[here])
	assert.False(t, found[there])
	require.NotNil(t, result.TotalCount)
	assert.Equal(t, len(result.Items), *result.TotalCount)

	everywhere, err := repo.ListConnection(
		ctx,
		&servicefailureports.ListServiceFailureConnectionRequest{
			Filter: &pagination.QueryOptions{TenantInfo: tenant},
			Cursor: pagination.CursorInfo{Limit: 100, IncludeTotalCount: true},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, everywhere.TotalCount)
	assert.Greater(t, *everywhere.TotalCount, *result.TotalCount)
}

func TestListConnection_FollowsAStopToItsNewLocation(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	st := shipmentdb.Stops
	first, err := st.AllTenants().Order(st.ID.Asc()).First(dbscope.WithSystem(ctx, "test"), db)
	require.NoError(t, err)
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: first.OrganizationID,
		BusinessUnitID: first.BusinessUnitID,
	})

	candidates, err := st.Order(st.ID.Asc()).Limit(200).List(ctx, db)
	require.NoError(t, err)
	stops := atDistinctLocations(candidates, 2)
	require.Len(t, stops, 2, "the development seed should hold stops at two locations")

	moved := recordFailureAt(ctx, t, db, stops[0])
	from, to := stops[0].LocationID, stops[1].LocationID
	_, err = st.Where(st.ID.Eq(stops[0].ID)).Update(ctx, db, st.LocationID.To(to))
	require.NoError(t, err)

	repo := servicefailurerepository.New(servicefailurerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: stops[0].OrganizationID, BuID: stops[0].BusinessUnitID}
	atLocation := func(location pulid.ID) map[pulid.ID]bool {
		t.Helper()
		result, listErr := repo.ListConnection(
			ctx,
			&servicefailureports.ListServiceFailureConnectionRequest{
				Filter:     &pagination.QueryOptions{TenantInfo: tenant},
				Cursor:     pagination.CursorInfo{Limit: 100},
				LocationID: &location,
			},
		)
		require.NoError(t, listErr)
		found := make(map[pulid.ID]bool, len(result.Items))
		for _, failure := range result.Items {
			found[failure.ID] = true
		}
		return found
	}

	assert.True(t, atLocation(to)[moved], "the failure moves with its stop")
	assert.False(t, atLocation(from)[moved], "the failure leaves the stop's old location")
}
