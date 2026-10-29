package shipmentprovider

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetentionToneAndLabelFollowTheDwell(t *testing.T) {
	t.Parallel()

	long := int64(4*60*60 + 1)
	short := int64(2*60*60 + 30*60)

	assert.Equal(t, "danger", detentionTone(long))
	assert.Equal(t, "warning", detentionTone(short))
	assert.Equal(t, "4h 00m", formatDwell(long))
	assert.Equal(t, "2h 30m", formatDwell(short))
}

func TestBuildLaneHeatmapCardAggregatesRegions(t *testing.T) {
	t.Parallel()

	card := buildLaneHeatmapCard(7, []laneStateRow{
		{OriginState: "CA", DestinationState: "TX", Count: 4},
		{OriginState: "CA", DestinationState: "WA", Count: 3},
		{OriginState: "PR", DestinationState: "TX", Count: 9},
		{OriginState: "NY", DestinationState: "PR", Count: 7},
	})

	require.NotNil(t, card)
	assert.Equal(t, 7, card.WindowDays)
	assert.Equal(t, 7, card.Total)
	assert.Contains(t, card.Cells, &LaneHeatmapCell{Origin: "West", Destination: "South", Count: 4})
	assert.Contains(t, card.Cells, &LaneHeatmapCell{Origin: "West", Destination: "West", Count: 3})
	assert.NotContains(
		t,
		card.Cells,
		&LaneHeatmapCell{Origin: "South", Destination: "West", Count: 0},
	)
}

func TestBuildCustomerMixCardRanksByRevenueAndCalculatesMetrics(t *testing.T) {
	t.Parallel()

	card := buildCustomerMixCard([]customerMixRow{
		{
			CustomerID:      "cus_1",
			Name:            "Acme Manufacturing",
			Revenue:         40000,
			Loads:           10,
			PreviousRevenue: 20000,
			TotalRevenue:    50000,
		},
		{
			CustomerID:   "cus_2",
			Name:         "FreshHaul Foods",
			Revenue:      10000,
			Loads:        4,
			TotalRevenue: 50000,
		},
	})

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
}

func TestBuildTomorrowsPickupsCardMapsStatuses(t *testing.T) {
	t.Parallel()

	tomorrow := time.Date(2026, time.May, 8, 0, 0, 0, 0, time.UTC)
	pickupStart := time.Date(2026, time.May, 8, 9, 30, 0, 0, time.UTC).Unix()

	card := buildTomorrowsPickupsCard(tomorrow, []tomorrowPickupRow{
		{
			ShipmentID: "sp_1", ProNumber: "PRO-1001", PickupWindowStart: pickupStart,
			Customer: "Acme Manufacturing", Origin: "TERM-LA", Destination: "DC-CHI",
			Driver: "M. Alvarez", ShipmentStatus: shipmentvocab.StatusAssigned, HasPrimaryWorker: true,
		},
		{
			ShipmentID: "sp_2", ProNumber: "PRO-1002", PickupWindowStart: pickupStart + 3600,
			Customer: "FreshHaul Foods", Origin: "COLD-SEA", Destination: "DC-PDX",
			ShipmentStatus: shipmentvocab.StatusNew,
		},
		{
			ShipmentID: "sp_3", ProNumber: "PRO-1003", PickupWindowStart: pickupStart + 7200,
			Customer: "Range Logistics", Origin: "ATL", Destination: "CLT",
			Driver: "A. Romero", ShipmentStatus: shipmentvocab.StatusPartiallyAssigned, HasPrimaryWorker: true,
		},
	})

	require.NotNil(t, card)
	require.Len(t, card.Pickups, 3)
	assert.Equal(t, "2026-05-08", card.Date)
	assert.Equal(t, TomorrowPickupStatusConfirmed, card.Pickups[0].Status)
	assert.Equal(t, TomorrowPickupStatusUnassigned, card.Pickups[1].Status)
	assert.Equal(t, TomorrowPickupStatusTentative, card.Pickups[2].Status)
	assert.Equal(t, new("TERM-LA"), card.Pickups[0].Origin)
	assert.Equal(t, new("DC-CHI"), card.Pickups[0].Destination)
}

func TestAnalyticsQueriesRender(t *testing.T) {
	t.Parallel()

	window := analyticsWindow{
		TodayStart:     1_000,
		TomorrowStart:  2_000,
		Now:            1_500,
		YesterdayStart: 0,
		YesterdayEnd:   500,
		SevenDayStart:  -6_000,
	}

	cases := map[string]keeltest.Renderer{
		"active_shipments": activeShipmentsQuery(window),
		"hourly_shipments": hourlyShipmentsQuery(window, "America/Chicago"),
		"on_time":          onTimeQuery(window),
		"revenue_today":    revenueTodayQuery(window),
		"hourly_revenue":   hourlyRevenueQuery(window, "America/Chicago"),
		"empty_mile":       emptyMileQuery(window),
		"at_risk":          atRiskQuery(window),
		"unassigned":       unassignedQuery(window),
		"ready_dispatch":   readyToDispatchQuery(window),
		"detention":        detentionWatchlistQuery(1_500),
		"customer_mix":     customerMixQuery(1_500, 1_000, 500),
		"tomorrow_pickups": tomorrowsPickupsQuery(2_000, 3_000, 20, 0),
	}
	for name, q := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			keeltest.Snapshot(t, q)
		})
	}
}
