//go:build integration

package servicefailurerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurerepository"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type stopAtLocation struct {
	StopID         pulid.ID `bun:"stop_id"`
	LocationID     pulid.ID `bun:"location_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func recordFailureAt(ctx context.Context, t *testing.T, db bun.IDB, stopID pulid.ID) pulid.ID {
	t.Helper()

	id := pulid.MustNew("sf_")
	_, err := db.NewRaw(`
		INSERT INTO service_failures (
			id, organization_id, business_unit_id, shipment_id, shipment_move_id, stop_id,
			number, type, stop_type, scheduled_cutoff, actual_arrival, late_minutes
		)
		SELECT ?, s.organization_id, s.business_unit_id, sm.shipment_id, s.shipment_move_id, s.id,
			?, 'LateDelivery', s.type, 1000, 4600, 60
		FROM stops AS s
		JOIN shipment_moves AS sm
			ON sm.id = s.shipment_move_id
			AND sm.organization_id = s.organization_id
			AND sm.business_unit_id = s.business_unit_id
		WHERE s.id = ?`,
		id, "SF-"+id.String()[len(id.String())-8:], stopID,
	).Exec(ctx)
	require.NoError(t, err)

	return id
}

func TestListConnection_NarrowsToFailuresAtTheLocationAsked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true})
	require.NoError(t, err)

	var stops []stopAtLocation
	require.NoError(t, db.NewRaw(`
		SELECT DISTINCT ON (s.location_id)
			s.id AS stop_id, s.location_id, s.organization_id, s.business_unit_id
		FROM stops AS s
		WHERE s.organization_id = (SELECT organization_id FROM stops ORDER BY id LIMIT 1)
			AND s.business_unit_id = (SELECT business_unit_id FROM stops ORDER BY id LIMIT 1)
		ORDER BY s.location_id, s.id
		LIMIT 2`).Scan(ctx, &stops))
	require.Len(t, stops, 2, "the development seed should hold stops at two locations")

	here := recordFailureAt(ctx, t, db, stops[0].StopID)
	there := recordFailureAt(ctx, t, db, stops[1].StopID)

	repo := servicefailurerepository.New(servicefailurerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: stops[0].OrganizationID, BuID: stops[0].BusinessUnitID}
	at := stops[0].LocationID
	result, err := repo.ListConnection(ctx, &servicefailureports.ListServiceFailureConnectionRequest{
		Filter:     &pagination.QueryOptions{TenantInfo: tenant},
		Cursor:     pagination.CursorInfo{Limit: 100, IncludeTotalCount: true},
		LocationID: &at,
	})
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

	everywhere, err := repo.ListConnection(ctx, &servicefailureports.ListServiceFailureConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: tenant},
		Cursor: pagination.CursorInfo{Limit: 100, IncludeTotalCount: true},
	})
	require.NoError(t, err)
	require.NotNil(t, everywhere.TotalCount)
	assert.Greater(t, *everywhere.TotalCount, *result.TotalCount)
}
