//go:build integration

package shipmentmovejurisdictionmilerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentmovejurisdictionmilerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const completedAt = int64(1_800_000_000)

type fixture struct {
	ctx        context.Context
	db         *bun.DB
	repo       repositories.ShipmentMoveJurisdictionMileRepository
	tenantInfo pagination.TenantInfo
	shipmentID pulid.ID
	locationID pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)

	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "integration-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("organizations").
		Column("id", "business_unit_id").
		Order("created_at ASC").
		Limit(1).
		Scan(ctx, &org))
	tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	var seeded struct {
		ShipmentID pulid.ID `bun:"shipment_id"`
		LocationID pulid.ID `bun:"location_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("stops AS st").
		ColumnExpr("sm.shipment_id").
		ColumnExpr("st.location_id").
		Join("JOIN shipment_moves AS sm ON sm.id = st.shipment_move_id").
		Where("st.organization_id = ?", tenantInfo.OrgID).
		Where("st.business_unit_id = ?", tenantInfo.BuID).
		Limit(1).
		Scan(ctx, &seeded))

	return &fixture{
		ctx: ctx,
		db:  db,
		repo: shipmentmovejurisdictionmilerepository.New(
			shipmentmovejurisdictionmilerepository.Params{
				DB:     postgres.NewTestConnection(db),
				Logger: zap.NewNop(),
			},
		),
		tenantInfo: tenantInfo,
		shipmentID: seeded.ShipmentID,
		locationID: seeded.LocationID,
	}
}

func (f *fixture) insertCompletedMove(
	t *testing.T,
	distance float64,
	departedAt int64,
) *shipment.ShipmentMove {
	t.Helper()

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: f.tenantInfo.BuID,
		OrganizationID: f.tenantInfo.OrgID,
		ShipmentID:     f.shipmentID,
		Status:         shipment.MoveStatusCompleted,
		Loaded:         true,
		Sequence:       90,
		Distance:       &distance,
		DistanceUnits:  shipment.JurisdictionDistanceUnitsMiles,
	}
	_, err := f.db.NewInsert().Model(move).Exec(f.ctx)
	require.NoError(t, err)

	arrivedAt := departedAt - 3600
	stops := []*shipment.Stop{
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       f.tenantInfo.BuID,
			OrganizationID:       f.tenantInfo.OrgID,
			ShipmentMoveID:       move.ID,
			LocationID:           f.locationID,
			Status:               shipment.StopStatusCompleted,
			Type:                 shipment.StopTypePickup,
			Sequence:             0,
			ScheduledWindowStart: arrivedAt - 7200,
			ActualArrival:        &arrivedAt,
			ActualDeparture:      &departedAt,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       f.tenantInfo.BuID,
			OrganizationID:       f.tenantInfo.OrgID,
			ShipmentMoveID:       move.ID,
			LocationID:           f.locationID,
			Status:               shipment.StopStatusCompleted,
			Type:                 shipment.StopTypeDelivery,
			Sequence:             1,
			ScheduledWindowStart: arrivedAt,
			ActualArrival:        &departedAt,
		},
	}
	_, err = f.db.NewInsert().Model(&stops).Exec(f.ctx)
	require.NoError(t, err)

	return move
}

func row(country, code string, distance float64) *shipment.ShipmentMoveJurisdictionMile {
	return &shipment.ShipmentMoveJurisdictionMile{
		CountryCode:      country,
		JurisdictionCode: code,
		Distance:         distance,
		DistanceUnits:    shipment.JurisdictionDistanceUnitsMiles,
		Loaded:           true,
		Source:           shipment.JurisdictionMileSourceRouteCalculation,
		CalculatedAt:     completedAt,
	}
}

func TestReplaceForMoveReplacesRows(t *testing.T) {
	f := setup(t)
	move := f.insertCompletedMove(t, 150, completedAt)

	move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{
		row("US", "TX", 100),
		row("US", "OK", 50),
	}
	require.NoError(t, f.repo.ReplaceForMove(f.ctx, move))
	assert.False(t, move.JurisdictionMilesDirty)

	rows, err := f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: f.tenantInfo,
		MoveIDs:    []pulid.ID{move.ID},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "TX", rows[0].JurisdictionCode)
	assert.Equal(t, move.ID, rows[0].ShipmentMoveID)
	assert.Equal(t, f.shipmentID, rows[0].ShipmentID)
	assert.Equal(t, f.tenantInfo.OrgID, rows[0].OrganizationID)
	assert.False(t, rows[0].ID.IsNil())

	move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{row("US", "NM", 30)}
	require.NoError(t, f.repo.ReplaceForMove(f.ctx, move))

	rows, err = f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: f.tenantInfo,
		MoveIDs:    []pulid.ID{move.ID},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "NM", rows[0].JurisdictionCode)

	move.JurisdictionMiles = nil
	require.NoError(t, f.repo.ReplaceForMove(f.ctx, move))

	rows, err = f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: f.tenantInfo,
		MoveIDs:    []pulid.ID{move.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestListByMoveIDsIsTenantIsolated(t *testing.T) {
	f := setup(t)
	move := f.insertCompletedMove(t, 150, completedAt)
	move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{row("US", "TX", 150)}
	require.NoError(t, f.repo.ReplaceForMove(f.ctx, move))

	rows, err := f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		MoveIDs:    []pulid.ID{move.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, rows)

	rows, err = f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: f.tenantInfo,
		MoveIDs:    []pulid.ID{move.ID},
	})
	require.NoError(t, err)
	assert.Len(t, rows, 1)

	rows, err = f.repo.ListByMoveIDs(f.ctx, repositories.ListJurisdictionMilesByMoveIDsRequest{
		TenantInfo: f.tenantInfo,
	})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestListUnattributedMovesCountsCompletedMovesInPeriod(t *testing.T) {
	f := setup(t)
	move := f.insertCompletedMove(t, 120, completedAt)

	page, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt - 100,
		End:        completedAt + 100,
	})
	require.NoError(t, err)
	assert.Contains(t, page.MoveIDs, move.ID)
	assert.GreaterOrEqual(t, page.TotalMoves, 1)
	assert.GreaterOrEqual(t, page.TotalMiles.InexactFloat64(), 120.0)

	before, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt - 100,
		End:        completedAt,
	})
	require.NoError(t, err)
	assert.NotContains(t, before.MoveIDs, move.ID)

	after, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt + 1,
		End:        completedAt + 100,
	})
	require.NoError(t, err)
	assert.NotContains(t, after.MoveIDs, move.ID)

	otherTenant, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Start:      completedAt - 100,
		End:        completedAt + 100,
	})
	require.NoError(t, err)
	assert.Empty(t, otherTenant.MoveIDs)
	assert.Equal(t, 0, otherTenant.TotalMoves)

	move.JurisdictionMiles = []*shipment.ShipmentMoveJurisdictionMile{row("US", "TX", 120)}
	require.NoError(t, f.repo.ReplaceForMove(f.ctx, move))

	attributed, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt - 100,
		End:        completedAt + 100,
	})
	require.NoError(t, err)
	assert.NotContains(t, attributed.MoveIDs, move.ID)
	assert.Equal(t, page.TotalMoves-1, attributed.TotalMoves)

	_, err = f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt,
		End:        completedAt,
	})
	require.Error(t, err)
}

func TestListUnattributedMovesSkipsMovesWithoutDistance(t *testing.T) {
	f := setup(t)
	move := f.insertCompletedMove(t, 0, completedAt)

	page, err := f.repo.ListUnattributedMoves(f.ctx, repositories.UnattributedMovesRequest{
		TenantInfo: f.tenantInfo,
		Start:      completedAt - 100,
		End:        completedAt + 100,
	})
	require.NoError(t, err)
	assert.NotContains(t, page.MoveIDs, move.ID)
}
