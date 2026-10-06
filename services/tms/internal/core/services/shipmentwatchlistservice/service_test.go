package shipmentwatchlistservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func ptr[T any](v T) *T { return &v }

func TestBuildDeliveryWatch(t *testing.T) {
	t.Parallel()

	loc := time.UTC
	day := time.Date(2026, 10, 6, 0, 0, 0, 0, loc).Unix()
	now := day + 14*3600

	rows := []*repositories.ShipmentDeliveryRow{
		{
			ShipmentID:    pulid.MustNew("shp_"),
			DeliveryAt:    day + 8*3600,
			ActualArrival: ptr(day + 8*3600),
			Cutoff:        day + 9*3600,
		},
		{
			ShipmentID:    pulid.MustNew("shp_"),
			ProNumber:     "P2",
			DeliveryAt:    day + 9*3600,
			ActualArrival: ptr(day + 10*3600),
			Cutoff:        day + 9*3600,
		},
		{
			ShipmentID: pulid.MustNew("shp_"),
			ProNumber:  "P3",
			DeliveryAt: day + 11*3600,
			Cutoff:     day + 11*3600,
		},
		{ShipmentID: pulid.MustNew("shp_"), DeliveryAt: day + 16*3600, Cutoff: day + 16*3600},
		{
			ShipmentID: pulid.MustNew("shp_"),
			DeliveryAt: day + 17*3600,
			Cutoff:     day + 17*3600,
			StageRank:  shipment.StageLate.Rank(),
		},
		{
			ShipmentID:    pulid.MustNew("shp_"),
			DeliveryAt:    day + 2*3600,
			ActualArrival: ptr(day + 2*3600),
			Cutoff:        day + 3*3600,
		},
	}

	watch := BuildDeliveryWatch(rows, now, loc)

	assert.Equal(t, 6, watch.Total)
	assert.Equal(t, 2, watch.OnTime)
	assert.Equal(t, 3, watch.LateCount)
	require.Len(t, watch.Buckets, 18)
	assert.Equal(t, 6, watch.Buckets[0].Hour)
	assert.Equal(t, 23, watch.Buckets[17].Hour)
	assert.Equal(t, 1, watch.Buckets[8-6].Delivered)
	assert.Equal(t, 1, watch.Buckets[9-6].Late)
	assert.Equal(t, 1, watch.Buckets[11-6].Late)
	assert.Equal(t, 1, watch.Buckets[16-6].Scheduled)
	assert.Equal(t, 1, watch.Buckets[17-6].Late)
	require.Len(t, watch.WorstLate, 3)
	assert.Equal(t, "P3", watch.WorstLate[0].ProNumber)
	assert.Equal(t, 180, watch.WorstLate[0].DeltaMinutes)
	assert.Equal(t, "P2", watch.WorstLate[1].ProNumber)
	assert.Equal(t, 60, watch.WorstLate[1].DeltaMinutes)
}

func TestAccruedAmountAndSummary(t *testing.T) {
	t.Parallel()

	rate := decimal.NewFromInt(60)
	assert.True(t, AccruedAmount(rate, 1000, 1000+5400).Equal(decimal.NewFromInt(90)))
	assert.True(t, AccruedAmount(rate, 1000, 900).IsZero())
	assert.True(t, AccruedAmount(decimal.Zero, 0, 3600).IsZero())

	accruals := []services.DetentionAccrual{
		{FacilityName: "a", Amount: decimal.NewFromInt(10), RatePerHour: decimal.NewFromInt(50)},
		{FacilityName: "b", Amount: decimal.NewFromInt(40), RatePerHour: decimal.NewFromInt(50)},
		{FacilityName: "c", Amount: decimal.NewFromInt(30), RatePerHour: decimal.NewFromInt(75)},
		{FacilityName: "d", Amount: decimal.NewFromInt(5), RatePerHour: decimal.Zero},
	}
	watch := SummarizeDetention(accruals, 99)
	assert.Equal(t, 4, watch.StopCount)
	assert.True(t, watch.Amount.Equal(decimal.NewFromInt(85)))
	assert.True(t, watch.RatePerHour.Equal(decimal.NewFromInt(175)))
	assert.Equal(t, int64(99), watch.SnapshotAt)
	require.Len(t, watch.Top, 3)
	assert.Equal(t, "b", watch.Top[0].FacilityName)
	assert.Equal(t, "c", watch.Top[1].FacilityName)
}

type stubBasis struct{ now time.Time }

func (s stubBasis) Resolve(
	context.Context,
	*services.ResolveShipmentQuickFilterBasisRequest,
) (*repositories.ShipmentQuickFilterBasis, error) {
	return &repositories.ShipmentQuickFilterBasis{Now: s.now, Location: time.UTC}, nil
}

func (stubBasis) Prepare(
	context.Context,
	pagination.TenantInfo,
	*repositories.ShipmentOptions,
) error {
	return nil
}

type stubBoard struct {
	repositories.ShipmentBoardRepository
	filters []shipment.QuickFilterSpec
}

func (s *stubBoard) QuickFilterTotals(
	_ context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) ([]repositories.ShipmentQuickFilterTotal, error) {
	s.filters = req.Filters
	totals := make([]repositories.ShipmentQuickFilterTotal, len(req.Filters))
	for i := range totals {
		totals[i] = repositories.ShipmentQuickFilterTotal{
			Count:   i + 1,
			Revenue: decimal.NewFromInt(int64(100 * (i + 1))),
		}
	}
	return totals, nil
}

type stubWatchlist struct {
	dwelling []*repositories.ShipmentDwellRow
}

func (s *stubWatchlist) ListDeliveriesToday(
	context.Context,
	*repositories.ShipmentWatchlistRequest,
) ([]*repositories.ShipmentDeliveryRow, error) {
	return nil, nil
}

func (s *stubWatchlist) NextUncoveredPickup(
	context.Context,
	*repositories.ShipmentWatchlistRequest,
) (*repositories.ShipmentPickupRow, error) {
	return &repositories.ShipmentPickupRow{PickupAt: 77, OriginCity: "Joliet"}, nil
}

func (s *stubWatchlist) ListDwellingStops(
	context.Context,
	*repositories.ShipmentWatchlistRequest,
) ([]*repositories.ShipmentDwellRow, error) {
	return s.dwelling, nil
}

func (s *stubWatchlist) ListReadyToBillCustomers(
	context.Context,
	*repositories.ListReadyToBillCustomersRequest,
) ([]*repositories.ShipmentBillingCustomerRow, error) {
	return []*repositories.ShipmentBillingCustomerRow{
		{Name: "Acme", Count: 2, Total: decimal.NewFromInt(500), TotalCustomers: 6},
	}, nil
}

type stubControls struct {
	repositories.ShipmentControlRepository
	control *tenant.ShipmentControl
}

func (s stubControls) Get(
	context.Context,
	repositories.GetShipmentControlRequest,
) (*tenant.ShipmentControl, error) {
	return s.control, nil
}

type stubAccessorials struct {
	repositories.AccessorialChargeRepository
}

func (stubAccessorials) GetByID(
	context.Context,
	repositories.GetAccessorialChargeByIDRequest,
) (*accessorialcharge.AccessorialCharge, error) {
	return &accessorialcharge.AccessorialCharge{
		Amount:   decimal.NewFromInt(60),
		RateUnit: accessorialcharge.RateUnitHour,
	}, nil
}

type stubMoves struct{}

func (stubMoves) ListBoardMoves(
	_ context.Context,
	filter *repositories.DispatchBoardFilter,
) ([]*repositories.BoardMove, error) {
	out := make([]*repositories.BoardMove, 0, len(filter.MoveIDs))
	for _, id := range filter.MoveIDs {
		out = append(out, &repositories.BoardMove{MoveID: id, AssignedCarrierName: "Swift"})
	}
	return out, nil
}

type stubDesk struct{ entries []*detentionservice.DeskEntry }

func (s stubDesk) ListDesk(
	context.Context,
	pagination.TenantInfo,
) ([]*detentionservice.DeskEntry, error) {
	return s.entries, nil
}

func TestWatchlistComputesEverySection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC)
	chargeID := pulid.MustNew("acc_")
	board := &stubBoard{}
	svc := NewWithDependencies(&Dependencies{
		Board: board,
		Watchlist: &stubWatchlist{dwelling: []*repositories.ShipmentDwellRow{{
			MoveID:        pulid.MustNew("smv_"),
			FacilityName:  "DC 4",
			ActualArrival: now.Unix() - 2*3600,
		}}},
		QuickFilters: stubBasis{now: now},
		Controls: stubControls{control: &tenant.ShipmentControl{
			DetentionThreshold: ptr(int16(30)),
			DetentionChargeID:  &chargeID,
		}},
		Accessorials: stubAccessorials{},
		Moves:        stubMoves{},
		Detention:    stubDesk{},
		Logger:       zap.NewNop(),
	})

	watch, err := svc.Watchlist(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)

	assert.Equal(t, 1, watch.Uncovered.Count)
	require.Len(t, watch.Uncovered.Windows, 4)
	assert.Equal(t, shipment.PickupWindowUnderTwoHours, watch.Uncovered.Windows[0].Window)
	assert.Equal(t, 3, watch.Uncovered.Windows[0].Count)
	assert.Equal(t, "Joliet", watch.Uncovered.Next.OriginCity)
	assert.Equal(t, 2, watch.Billing.Count)
	assert.Equal(t, 5, watch.Billing.MoreCustomers)
	assert.Equal(t, shipment.QuickFilterUncovered, board.filters[0].Filter)
	assert.Equal(t, shipment.QuickFilterReadyToBill, board.filters[1].Filter)
	assert.Equal(t, shipment.QuickFilterPickupWindow, board.filters[2].Filter)

	require.Equal(t, 1, watch.Detention.StopCount)
	assert.True(t, watch.Detention.Amount.Equal(decimal.NewFromInt(90)))
	assert.Equal(t, "Swift", watch.Detention.Top[0].CoverageName)
	assert.Nil(t, watch.Detention.Top[0].OccurrenceID)
	assert.Len(t, watch.Deliveries.Buckets, 18)
}

func TestWatchlistUsesTheDetentionEngine(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 6, 15, 0, 0, 0, time.UTC)
	occurrenceID := pulid.MustNew("dto_")
	svc := NewWithDependencies(&Dependencies{
		Board:        &stubBoard{},
		Watchlist:    &stubWatchlist{},
		QuickFilters: stubBasis{now: now},
		Controls: stubControls{control: &tenant.ShipmentControl{
			UseDetentionPolicyEngine: true,
		}},
		Accessorials: stubAccessorials{},
		Moves:        stubMoves{},
		Detention: stubDesk{entries: []*detentionservice.DeskEntry{
			{Occurrence: &detention.DetentionOccurrence{
				ID:                occurrenceID,
				FreeTimeExpiresAt: now.Unix() - 3600,
				PolicySnapshot: &detention.PolicySnapshot{
					FlatRate:     decimal.NewFromInt(80),
					FlatRateUnit: detention.TierRateUnitHour,
				},
			}},
			{Occurrence: &detention.DetentionOccurrence{FreeTimeExpiresAt: now.Unix() + 600}},
		}},
		Logger: zap.NewNop(),
	})

	watch, err := svc.Watchlist(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)
	require.Equal(t, 1, watch.Detention.StopCount)
	assert.Equal(t, occurrenceID, *watch.Detention.Top[0].OccurrenceID)
	assert.True(t, watch.Detention.Amount.Equal(decimal.NewFromInt(80)))
}
