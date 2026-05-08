package shipmentprovider

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestProvider(t *testing.T) (*Provider, sqlmock.Sqlmock, *mocks.MockShipmentControlRepository) {
	t.Helper()

	db, mockDB, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mockDB.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	controlRepo := mocks.NewMockShipmentControlRepository(t)

	return &Provider{
		l:           zap.NewNop(),
		db:          postgres.NewTestConnection(bunDB),
		controlRepo: controlRepo,
	}, mockDB, controlRepo
}

func TestGetDetentionAlerts_ReturnsZeroWhenTrackingDisabled(t *testing.T) {
	t.Parallel()

	provider, mockDB, controlRepo := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().
		Get(t.Context(), repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{TrackDetentionTime: false}, nil).
		Once()

	card, err := provider.getDetentionAlerts(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, 0, card.Count)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetDetentionAlerts_UsesShipmentControlThreshold(t *testing.T) {
	t.Parallel()

	provider, mockDB, controlRepo := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	controlRepo.EXPECT().
		Get(t.Context(), repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&tenant.ShipmentControl{
			TrackDetentionTime: true,
			DetentionThreshold: ptrInt16(45),
		}, nil).
		Once()

	mockDB.ExpectQuery(`SELECT count\(\*\) FROM stops stp.*\(stp\.actual_departure - stp\.actual_arrival\) > .*2700.*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	card, err := provider.getDetentionAlerts(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, 3, card.Count)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetLaneHeatmap_AggregatesRegions(t *testing.T) {
	t.Parallel()

	provider, mockDB, _ := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockDB.ExpectQuery(`(?s)WITH shipment_lanes AS .*SELECT origin_state, destination_state, COUNT\(\*\)::int AS count`).
		WillReturnRows(sqlmock.NewRows([]string{"origin_state", "destination_state", "count"}).
			AddRow("CA", "TX", 4).
			AddRow("CA", "WA", 3).
			AddRow("PR", "TX", 9).
			AddRow("NY", "PR", 7))

	card, err := provider.getLaneHeatmap(t.Context(), orgID, buID, 7)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, 7, card.WindowDays)
	assert.Equal(t, 7, card.Total)
	assert.Contains(t, card.Cells, &LaneHeatmapCell{
		Origin:      "West",
		Destination: "South",
		Count:       4,
	})
	assert.Contains(t, card.Cells, &LaneHeatmapCell{
		Origin:      "West",
		Destination: "West",
		Count:       3,
	})
	assert.NotContains(t, card.Cells, &LaneHeatmapCell{
		Origin:      "South",
		Destination: "West",
		Count:       0,
	})
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetCustomerMix_RanksByRevenueAndCalculatesMetrics(t *testing.T) {
	t.Parallel()

	provider, mockDB, _ := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockDB.ExpectQuery(`(?s)WITH customer_revenue AS .*ORDER BY revenue DESC\s+LIMIT 5`).
		WillReturnRows(sqlmock.NewRows([]string{
			"customer_id",
			"name",
			"revenue",
			"loads",
			"previous_revenue",
			"total_revenue",
		}).
			AddRow("cus_1", "Acme Manufacturing", 40000.0, 10, 20000.0, 50000.0).
			AddRow("cus_2", "FreshHaul Foods", 10000.0, 4, 0.0, 50000.0))

	card, err := provider.getCustomerMix(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, customerMixWindowDays, card.WindowDays)
	require.Len(t, card.Entries, 2)
	assert.Equal(t, &CustomerMixEntry{
		CustomerID: "cus_1",
		Name:       "Acme Manufacturing",
		Revenue:    40000,
		Share:      80,
		Loads:      10,
		Trend:      100,
	}, card.Entries[0])
	assert.Equal(t, &CustomerMixEntry{
		CustomerID: "cus_2",
		Name:       "FreshHaul Foods",
		Revenue:    10000,
		Share:      20,
		Loads:      4,
		Trend:      100,
	}, card.Entries[1])
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetTomorrowsPickups_ReturnsPickupRowsWithStatusMapping(t *testing.T) {
	t.Parallel()

	provider, mockDB, _ := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	pickupStart := time.Date(2026, time.May, 8, 9, 30, 0, 0, time.UTC).Unix()

	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.id AS shipment_id.*stp\.status != .*sm\.status != .*sp\.status != .*ORDER BY stp\.scheduled_window_start ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"shipment_id",
			"pro_number",
			"pickup_window_start",
			"customer",
			"origin",
			"destination",
			"driver",
			"shipment_status",
			"has_primary_worker",
		}).
			AddRow("sp_1", "PRO-1001", pickupStart, "Acme Manufacturing", "TERM-LA", "DC-CHI", "M. Alvarez", shipment.StatusAssigned, true).
			AddRow("sp_2", "PRO-1002", pickupStart+3600, "FreshHaul Foods", "COLD-SEA", "DC-PDX", "", shipment.StatusNew, false).
			AddRow("sp_3", "PRO-1003", pickupStart+7200, "Range Logistics", "ATL", "CLT", "A. Romero", shipment.StatusPartiallyAssigned, true))

	card, err := provider.getTomorrowsPickups(t.Context(), orgID, buID, "UTC")

	require.NoError(t, err)
	require.NotNil(t, card)
	require.Len(t, card.Pickups, 3)
	assert.NotEmpty(t, card.Date)
	assert.Equal(t, TomorrowPickupStatusConfirmed, card.Pickups[0].Status)
	assert.Equal(t, TomorrowPickupStatusUnassigned, card.Pickups[1].Status)
	assert.Equal(t, TomorrowPickupStatusTentative, card.Pickups[2].Status)
	assert.Equal(t, "TERM-LA", card.Pickups[0].Origin)
	assert.Equal(t, "DC-CHI", card.Pickups[0].Destination)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

//go:fix inline
func ptrInt16(v int16) *int16 {
	return &v
}
