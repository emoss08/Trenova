package shipmentprovider

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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

func TestGetAnalyticsData_ReturnsSupportedShipmentKPIsOnly(t *testing.T) {
	t.Parallel()

	provider, mockDB, dispatchRepo := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	target := 96.5

	mockDB.ExpectQuery(`(?s)WITH shipment_lanes AS .*SELECT origin_state, destination_state, COUNT\(\*\)::int AS count`).
		WillReturnRows(sqlmock.NewRows([]string{"origin_state", "destination_state", "count"}))
	mockDB.ExpectQuery(`(?s)SELECT COUNT\(\*\) FILTER .* AS total_active.*FROM shipments sp`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_active",
			"created_today",
			"created_yesterday",
			"in_transit",
			"at_risk",
			"loading",
			"done",
		}).AddRow(8, 3, 1, 4, 1, 2, 1))
	mockDB.ExpectQuery(`(?s)SELECT EXTRACT\(HOUR FROM TO_TIMESTAMP\(sp\.created_at\).*COUNT\(\*\)::float8 AS value`).
		WillReturnRows(sqlmock.NewRows([]string{"hr", "value"}).AddRow(9, 2.0))
	mockDB.ExpectQuery(`(?s)WITH revenue AS .*SELECT revenue\.amount, revenue\.yesterday_amount, mileage\.miles`).
		WillReturnRows(sqlmock.NewRows([]string{"amount", "yesterday_amount", "miles"}).
			AddRow(1200.0, 1000.0, 600.0))
	mockDB.ExpectQuery(`(?s)SELECT EXTRACT\(HOUR FROM TO_TIMESTAMP\(sp\.actual_delivery_date\).*SUM\(sp\.total_charge_amount\).* AS value`).
		WillReturnRows(sqlmock.NewRows([]string{"hr", "value"}).AddRow(10, 1200.0))
	mockDB.ExpectQuery(`(?s)SELECT COUNT\(\*\) FILTER .* AS total.*FROM stops stp`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total",
			"on_time",
			"yesterday_total",
			"yesterday_on_time",
			"seven_day_total",
			"seven_day_on_time",
		}).AddRow(10, 9, 8, 7, 70, 63))

	dispatchRepo.EXPECT().
		GetByOrgID(t.Context(), repositories.GetDispatchControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		}).
		Return(&dispatchcontrol.DispatchControl{ServiceFailureTarget: &target}, nil).
		Once()

	mockDB.ExpectQuery(`(?s)SELECT COALESCE\(SUM\(sm\.distance\) FILTER .* AS total_miles.*FROM shipment_moves sm`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_miles",
			"empty_miles",
			"yesterday_total_miles",
			"yesterday_empty_miles",
		}).AddRow(1000.0, 100.0, 900.0, 120.0))
	mockDB.ExpectQuery(`(?s)WITH risky_shipments AS .*active_weather AS`).
		WillReturnRows(sqlmock.NewRows([]string{
			"count",
			"created_today",
			"created_yesterday",
			"eta_slip",
			"weather",
			"reefer",
		}).AddRow(2, 1, 0, 2, 1, 1))
	mockDB.ExpectQuery(`(?s)WITH unassigned_shipments AS`).
		WillReturnRows(sqlmock.NewRows([]string{
			"count",
			"created_today",
			"created_yesterday",
			"revenue_waiting",
		}).AddRow(3, 1, 2, 2500.0))
	mockDB.ExpectQuery(`(?s)WITH active_shipments AS .*shipment_assignments AS`).
		WillReturnRows(sqlmock.NewRows([]string{
			"count",
			"created_today",
			"created_yesterday",
			"unassigned",
			"driver_ready",
		}).AddRow(4, 2, 1, 3, 4))
	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.pro_number AS shipment_id.*ORDER BY dwell_seconds DESC.*LIMIT 10`).
		WillReturnRows(sqlmock.NewRows([]string{"shipment_id", "customer", "dwell_seconds"}))
	mockDB.ExpectQuery(`(?s)WITH customer_revenue AS .*ORDER BY revenue DESC\s+LIMIT 5`).
		WillReturnRows(sqlmock.NewRows([]string{
			"customer_id",
			"name",
			"revenue",
			"loads",
			"previous_revenue",
			"total_revenue",
		}))
	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.id AS shipment_id.*ORDER BY stp\.scheduled_window_start ASC.*LIMIT .*OFFSET`).
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
		}))

	data, err := provider.GetAnalyticsData(t.Context(), &services.AnalyticsRequestOptions{
		OrgID:    orgID,
		BuID:     buID,
		Timezone: "UTC",
	})

	require.NoError(t, err)
	assert.Contains(t, data, "revenueToday")
	assert.Contains(t, data, "activeShipments")
	assert.Contains(t, data, "detentionWatchlist")
	assert.NotContains(t, data, "tenderAccept")
	assert.NotContains(t, data, "hosNearLimit")
	assert.NotContains(t, data, "detentionAlerts")
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAnalyticsData_ReturnsSavedViewCountsInclude(t *testing.T) {
	t.Parallel()

	provider, mockDB, _ := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockDB.ExpectQuery(`(?s)SELECT\s+COUNT\(\*\) FILTER \(\s+WHERE sp\.organization_id = .*sp\.business_unit_id = .*\)::int AS "all".*COUNT\(\*\) FILTER \(\s+WHERE sp\.organization_id = .*sp\.business_unit_id = .*sp\.status = .*\)::int AS transit.*COUNT\(\*\) FILTER \(\s+WHERE sp\.organization_id = .*sp\.business_unit_id = .*sp\.status = .*\)::int AS at_risk.*COUNT\(\*\) FILTER \(\s+WHERE sp\.organization_id = .*sp\.business_unit_id = .*sp\.status IN .*\)::int AS unassigned.*EXISTS \(\s+SELECT 1\s+FROM shipment_moves sm\s+INNER JOIN stops stp.*AND sm\.sequence = \(\s+SELECT MAX\(sm2\.sequence\).*AND stp\.type IN .*AND stp\.schedule_type = .*AND stp\.scheduled_window_start >= .*AND stp\.scheduled_window_start < .*\)::int AS delivering_today\s+FROM shipments sp\s+WHERE sp\.organization_id = .*AND sp\.business_unit_id = .*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"all",
			"transit",
			"at_risk",
			"unassigned",
			"delivering_today",
		}).AddRow(11, 4, 2, 3, 5))

	data, err := provider.GetAnalyticsData(t.Context(), &services.AnalyticsRequestOptions{
		OrgID:    orgID,
		BuID:     buID,
		Timezone: "America/New_York",
		Include:  savedViewCountsInclude,
	})

	require.NoError(t, err)
	require.Len(t, data, 2)
	assert.Equal(t, string(services.ShipmentAnalyticsPage), data["page"])
	counts, ok := data["savedViewCounts"].(*SavedViewCounts)
	require.True(t, ok)
	assert.Equal(t, &SavedViewCounts{
		All:             11,
		Transit:         4,
		AtRisk:          2,
		Unassigned:      3,
		DeliveringToday: 5,
	}, counts)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func newTestProvider(t *testing.T) (*Provider, sqlmock.Sqlmock, *mocks.MockDispatchControlRepository) {
	t.Helper()

	db, mockDB, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mockDB.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	dispatchRepo := mocks.NewMockDispatchControlRepository(t)

	return &Provider{
		l:            zap.NewNop(),
		db:           postgres.NewTestConnection(bunDB),
		dispatchRepo: dispatchRepo,
	}, mockDB, dispatchRepo
}

func TestGetDetentionWatchlist_ReturnsRowsWithToneThresholds(t *testing.T) {
	t.Parallel()

	provider, mockDB, _ := newTestProvider(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.pro_number AS shipment_id.*ORDER BY dwell_seconds DESC.*LIMIT 10`).
		WillReturnRows(sqlmock.NewRows([]string{"shipment_id", "customer", "dwell_seconds"}).
			AddRow("SHP-1001", "Acme Manufacturing", int64(4*60*60+1)).
			AddRow("SHP-1002", "FreshHaul Foods", int64(2*60*60+30*60)))

	card, err := provider.getDetentionWatchlist(t.Context(), orgID, buID)

	require.NoError(t, err)
	require.NotNil(t, card)
	require.Len(t, card.Items, 2)
	assert.Equal(t, "danger", card.Items[0].Tone)
	assert.Equal(t, "warning", card.Items[1].Tone)
	assert.Equal(t, "4h 00m", card.Items[0].DwellLabel)
	assert.Equal(t, "2h 30m", card.Items[1].DwellLabel)
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

	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.id AS shipment_id.*stp\.status != .*sm\.status != .*sp\.status != .*ORDER BY stp\.scheduled_window_start ASC.*LIMIT .*OFFSET`).
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

	card, err := provider.getTomorrowsPickups(t.Context(), tomorrowsPickupsRequest{
		orgID:  orgID,
		buID:   buID,
		tz:     "UTC",
		limit:  20,
		offset: 0,
	})

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
