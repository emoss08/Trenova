//go:build integration

package detentionrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrepository"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type seededStop struct {
	StopID         pulid.ID `bun:"stop_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

type closedOccurrence struct {
	stopID       pulid.ID
	locationID   pulid.ID
	dwellMinutes int
	billed       string
}

func closeOccurrence(ctx context.Context, t *testing.T, db bun.IDB, occ closedOccurrence) {
	t.Helper()

	_, err := db.NewRaw(`
		INSERT INTO detention_occurrences (
			id, business_unit_id, organization_id, shipment_id, shipment_move_id, stop_id,
			customer_id, location_id, stop_type, schedule_type, clock_start_at,
			free_time_expires_at, is_open, raw_dwell_minutes, rounded_minutes, billable_amount,
			waived_amount, status, waiver_reason, waived_at
		)
		SELECT ?, s.business_unit_id, s.organization_id, sm.shipment_id, s.shipment_move_id, s.id,
			sp.customer_id, ?, s.type, s.schedule_type, 1000, 1060, FALSE, ?, ?, ?,
			?, 'Waived', 'Weather', 1100
		FROM stops AS s
		JOIN shipment_moves AS sm
			ON sm.id = s.shipment_move_id
			AND sm.organization_id = s.organization_id
			AND sm.business_unit_id = s.business_unit_id
		JOIN shipments AS sp
			ON sp.id = sm.shipment_id
			AND sp.organization_id = sm.organization_id
			AND sp.business_unit_id = sm.business_unit_id
		WHERE s.id = ?`,
		pulid.MustNew("dto_"),
		occ.locationID,
		occ.dwellMinutes,
		occ.dwellMinutes,
		occ.billed,
		occ.billed,
		occ.stopID,
	).Exec(ctx)
	require.NoError(t, err)
}

func TestDetentionStats_NarrowToTheLocationAsked(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true})
	require.NoError(t, err)

	var stops []seededStop
	require.NoError(t, db.NewRaw(`
		SELECT s.id AS stop_id, s.organization_id, s.business_unit_id
		FROM stops AS s
		WHERE NOT EXISTS (
			SELECT 1 FROM detention_occurrences AS d
			WHERE d.stop_id = s.id
				AND d.organization_id = s.organization_id
				AND d.business_unit_id = s.business_unit_id
		)
		ORDER BY s.organization_id, s.business_unit_id, s.id
		LIMIT 3`).Scan(ctx, &stops))
	require.Len(t, stops, 3, "the development seed should hold stops with no detention yet")
	require.Equal(t, stops[0].OrganizationID, stops[2].OrganizationID)
	require.Equal(t, stops[0].BusinessUnitID, stops[2].BusinessUnitID)

	at := pulid.MustNew("loc_")
	elsewhere := pulid.MustNew("loc_")
	closeOccurrence(ctx, t, db, closedOccurrence{stops[0].StopID, at, 90, "150.0000"})
	closeOccurrence(ctx, t, db, closedOccurrence{stops[1].StopID, at, 30, "40.0000"})
	closeOccurrence(ctx, t, db, closedOccurrence{stops[2].StopID, elsewhere, 600, "900.0000"})

	repo := detentionrepository.NewAnalyticsRepository(detentionrepository.AnalyticsParams{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	tenant := pagination.TenantInfo{
		OrgID: stops[0].OrganizationID,
		BuID:  stops[0].BusinessUnitID,
	}

	facilities, err := repo.FacilityStats(ctx, &detentionports.DetentionStatsRequest{
		TenantInfo: tenant,
		LocationID: at,
	})
	require.NoError(t, err)
	require.Len(t, facilities, 1)
	assert.Equal(t, at, facilities[0].LocationID)
	assert.Equal(t, 2, facilities[0].StopCount)
	assert.Equal(t, 60, facilities[0].AvgDwellMinutes)
	assert.Equal(t, "190", facilities[0].BilledAmount.String())

	everywhere, err := repo.FacilityStats(ctx, &detentionports.DetentionStatsRequest{
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	seen := make(map[pulid.ID]bool, len(everywhere))
	for _, facility := range everywhere {
		seen[facility.LocationID] = true
	}
	assert.True(t, seen[at])
	assert.True(t, seen[elsewhere])

	customers, err := repo.CustomerStats(ctx, &detentionports.DetentionStatsRequest{
		TenantInfo: tenant,
		LocationID: at,
	})
	require.NoError(t, err)
	stopCount := 0
	for _, customer := range customers {
		stopCount += customer.StopCount
	}
	assert.Equal(t, 2, stopCount)

	waivers, err := repo.WaiverStats(ctx, &detentionports.DetentionStatsRequest{
		TenantInfo: tenant,
		LocationID: at,
	})
	require.NoError(t, err)
	require.Len(t, waivers, 1)
	assert.Equal(t, 2, waivers[0].WaiverCount)
	assert.Equal(t, "190", waivers[0].WaivedAmount.String())
}
