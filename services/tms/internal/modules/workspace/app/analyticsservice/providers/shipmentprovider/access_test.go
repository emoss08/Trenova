package shipmentprovider

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readerEngine struct {
	mocks.AllowAllPermissionEngine

	denied  []permission.Resource
	ceiling permission.FieldSensitivity
}

func (e *readerEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: !slices.Contains(e.denied, permission.Resource(req.Resource)),
	}, nil
}

func (e *readerEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	if e.ceiling == "" {
		return nil, nil
	}
	return &permissionports.ResourcePermissionDetail{Resource: resource, MaxSensitivity: e.ceiling}, nil
}

type groupAccess struct {
	visible permission.FieldGroupSet
}

func (g groupAccess) Visible(context.Context, *permissionports.RequestActor) permission.FieldGroupSet {
	return g.visible
}

type reader struct {
	denied  []permission.Resource
	ceiling permission.FieldSensitivity
	margins bool
}

func (r reader) provider() *Provider {
	engine := &readerEngine{denied: r.denied, ceiling: r.ceiling}
	groups := permission.FieldGroupSet(0)
	if r.margins {
		groups = groups.With(permission.FieldGroupMargins)
	}
	return &Provider{
		views:       refview.NewPort(refview.New(permission.NewRegistry()), engine),
		fieldAccess: groupAccess{visible: groups},
	}
}

func (r reader) gate(t *testing.T, data map[string]any) {
	t.Helper()

	opts := &kernelports.AnalyticsRequestOptions{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	r.provider().accessFor(t.Context(), opts).gate(t.Context(), data)
}

var (
	withoutLinkedReads = []permission.Resource{
		permission.ResourceCustomer,
		permission.ResourceWorker,
		permission.ResourceLocation,
	}
	belowInternal = permission.SensitivityPublic
)

func customerMix() *CustomerMixCard {
	return buildCustomerMixCard([]customerMixRow{
		{CustomerID: "cus_1", Name: "Acme Foods", Revenue: 40000, Loads: 10, TotalRevenue: 50000},
		{CustomerID: "cus_2", Name: "FreshHaul", Revenue: 10000, Loads: 4, TotalRevenue: 50000},
	})
}

func TestCustomerMix_ShowsTheWholeCardToCustomerReaders(t *testing.T) {
	t.Parallel()

	card := customerMix()
	data := map[string]any{"customerMix": card}
	reader{denied: []permission.Resource{permission.ResourceWorker}}.gate(t, data)

	require.Same(t, card, data["customerMix"])
	require.Len(t, card.Entries, 2)
	assert.Equal(t, &CustomerMixEntry{
		CustomerID: "cus_1",
		Name:       "Acme Foods",
		Revenue:    40000,
		Share:      80,
		Loads:      10,
		Trend:      100,
	}, card.Entries[0])
}

func TestCustomerMix_HidesTheCardFromReadersWhoCannotReadCustomers(t *testing.T) {
	t.Parallel()

	for name, denied := range map[string][]permission.Resource{
		"without customer read":    {permission.ResourceCustomer},
		"without any linked reads": withoutLinkedReads,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := map[string]any{"customerMix": customerMix()}
			reader{denied: denied}.gate(t, data)

			assert.Nil(t, data["customerMix"],
				"the order of the names alone ranks the customers by revenue")
		})
	}
}

func TestCustomerMix_LeavesAnEmptyCardAlone(t *testing.T) {
	t.Parallel()

	data := map[string]any{"customerMix": (*CustomerMixCard)(nil)}
	reader{}.gate(t, data)

	assert.Nil(t, data["customerMix"])
}

func detentionWatchlist() *DetentionWatchlistCard {
	return &DetentionWatchlistCard{Items: []*DetentionWatchlistItem{
		{ShipmentID: "SHP-1001", Customer: new("Acme Foods"), DwellLabel: "4h 00m", Tone: "danger"},
	}}
}

func TestDetentionWatchlist_NamesTheCustomerThroughItsView(t *testing.T) {
	t.Parallel()

	card := detentionWatchlist()
	reader{denied: withoutLinkedReads}.gate(t, map[string]any{"detentionWatchlist": card})
	require.NotNil(t, card.Items[0].Customer)
	assert.Equal(t, "Acme Foods", *card.Items[0].Customer)

	held := detentionWatchlist()
	reader{denied: withoutLinkedReads, ceiling: belowInternal}.gate(t, map[string]any{"detentionWatchlist": held})
	assert.Nil(t, held.Items[0].Customer)
	assert.Equal(t, "SHP-1001", held.Items[0].ShipmentID)

	read := detentionWatchlist()
	reader{ceiling: belowInternal}.gate(t, map[string]any{"detentionWatchlist": read})
	assert.NotNil(t, read.Items[0].Customer, "the ceiling applies only without the kind's read")
}

func tomorrowsPickups() *TomorrowsPickupsCard {
	return buildTomorrowsPickupsCard(time.Date(2026, time.May, 8, 0, 0, 0, 0, time.UTC), []tomorrowPickupRow{{
		ShipmentID:       "sp_1",
		ProNumber:        "PRO-1001",
		Customer:         "Acme Foods",
		Origin:           "TERM-LA",
		Destination:      "DC-CHI",
		Driver:           "M. Alvarez",
		ShipmentStatus:   shipment.StatusAssigned,
		HasPrimaryWorker: true,
	}})
}

func TestTomorrowsPickups_ShowsLinkedRecordsThroughTheirViews(t *testing.T) {
	t.Parallel()

	card := tomorrowsPickups()
	reader{denied: withoutLinkedReads}.gate(t, map[string]any{"tomorrowsPickups": card})

	pickup := card.Pickups[0]
	require.NotNil(t, pickup.Customer)
	require.NotNil(t, pickup.Origin)
	require.NotNil(t, pickup.Destination)
	require.NotNil(t, pickup.Driver)
	assert.Equal(t, "M. Alvarez", *pickup.Driver)
}

func TestTomorrowsPickups_HoldsViewFieldsToTheCeiling(t *testing.T) {
	t.Parallel()

	card := tomorrowsPickups()
	reader{
		denied:  []permission.Resource{permission.ResourceWorker, permission.ResourceLocation},
		ceiling: belowInternal,
	}.gate(t, map[string]any{"tomorrowsPickups": card})

	pickup := card.Pickups[0]
	require.NotNil(t, pickup.Customer, "customer readers see the whole customer")
	assert.Nil(t, pickup.Origin)
	assert.Nil(t, pickup.Destination)
	assert.Nil(t, pickup.Driver)
	assert.Equal(t, "PRO-1001", pickup.ProNumber)
}

func TestLaneHeatmap_ShowsRegionsUnlessTheCeilingWithholdsAState(t *testing.T) {
	t.Parallel()

	shown := map[string]any{"laneHeatmap": buildLaneHeatmapCard(7, []laneStateRow{
		{OriginState: "CA", DestinationState: "TX", Count: 4},
	})}
	reader{denied: withoutLinkedReads}.gate(t, shown)
	require.NotNil(t, shown["laneHeatmap"])

	held := map[string]any{"laneHeatmap": buildLaneHeatmapCard(7, []laneStateRow{
		{OriginState: "CA", DestinationState: "TX", Count: 4},
	})}
	reader{denied: withoutLinkedReads, ceiling: belowInternal}.gate(t, held)
	assert.Nil(t, held["laneHeatmap"])
}

func TestProfitability_ShowsTheAverageMarginOnlyWithTheMarginsGroup(t *testing.T) {
	t.Parallel()

	shown := &ProfitabilityCard{AvgMarginPct: new(18.5), HasMargin: true, AvgCPM: 1.9}
	reader{margins: true}.gate(t, map[string]any{"profitability": shown})
	require.NotNil(t, shown.AvgMarginPct)
	assert.InDelta(t, 18.5, *shown.AvgMarginPct, 0.001)

	held := &ProfitabilityCard{AvgMarginPct: new(18.5), HasMargin: true, AvgCPM: 1.9}
	reader{}.gate(t, map[string]any{"profitability": held})
	assert.Nil(t, held.AvgMarginPct)
	assert.True(t, held.HasMargin)
	assert.InDelta(t, 1.9, held.AvgCPM, 0.001)
}

func TestGetAnalyticsData_GatesWhatItReturns(t *testing.T) {
	t.Parallel()

	deps := newTestProvider(t)
	provider, mockDB := deps.provider, deps.mockDB
	gated := reader{denied: withoutLinkedReads, ceiling: belowInternal}.provider()
	provider.views, provider.fieldAccess = gated.views, gated.fieldAccess

	mockDB.ExpectQuery(`(?s)SELECT\s+sp\.id AS shipment_id.*ORDER BY stp\.scheduled_window_start ASC.*LIMIT .*OFFSET`).
		WillReturnRows(sqlmock.NewRows([]string{
			"shipment_id", "pro_number", "pickup_window_start", "customer", "origin",
			"destination", "driver", "shipment_status", "has_primary_worker",
		}).AddRow("sp_1", "PRO-1001", int64(1), "Acme Foods", "TERM-LA", "DC-CHI", "M. Alvarez",
			shipment.StatusAssigned, true))

	data, err := provider.GetAnalyticsData(t.Context(), &kernelports.AnalyticsRequestOptions{
		OrgID:    pulid.MustNew("org_"),
		BuID:     pulid.MustNew("bu_"),
		UserID:   pulid.MustNew("usr_"),
		Timezone: "UTC",
		Include:  tomorrowsPickupsInclude,
	})
	require.NoError(t, err)

	card, ok := data["tomorrowsPickups"].(*TomorrowsPickupsCard)
	require.True(t, ok)
	require.Len(t, card.Pickups, 1)
	assert.Nil(t, card.Pickups[0].Customer)
	assert.Nil(t, card.Pickups[0].Driver)
	require.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAnalyticsData_WithholdsEveryLinkedValueWithoutTheViewsPort(t *testing.T) {
	t.Parallel()

	card := tomorrowsPickups()
	(&Provider{}).accessFor(t.Context(), &kernelports.AnalyticsRequestOptions{}).
		gate(t.Context(), map[string]any{"tomorrowsPickups": card})

	assert.Nil(t, card.Pickups[0].Customer)
	assert.Nil(t, card.Pickups[0].Driver)
}
