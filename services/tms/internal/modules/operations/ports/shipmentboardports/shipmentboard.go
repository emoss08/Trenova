package shipmentboardports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ShipmentBoardScope struct {
	Filter  *pagination.QueryOptions
	Options shipmentports.ShipmentOptions
}

type ShipmentStageSummaryRow struct {
	StageRank int16
	Count     int
	Revenue   decimal.Decimal
}

type ShipmentBoardGroupRow struct {
	Key     string
	Label   string
	Count   int
	Revenue decimal.Decimal
}

type SummarizeShipmentBoardGroupsQuery struct {
	Scope    *ShipmentBoardScope
	GroupBy  shipment.BoardGrouping
	Timezone string
}

type ShipmentQuickFilterTotal struct {
	Count   int
	Revenue decimal.Decimal
}

type CountShipmentQuickFiltersRequest struct {
	Scope   *ShipmentBoardScope
	Filters []shipment.QuickFilterSpec
}

type ShipmentBoardRepository interface {
	StageSummary(ctx context.Context, scope *ShipmentBoardScope) ([]*ShipmentStageSummaryRow, error)
	GroupSummary(
		ctx context.Context,
		req *SummarizeShipmentBoardGroupsQuery,
	) ([]*ShipmentBoardGroupRow, error)
	QuickFilterTotals(
		ctx context.Context,
		req *CountShipmentQuickFiltersRequest,
	) ([]ShipmentQuickFilterTotal, error)
}

type ShipmentWatchlistRequest struct {
	TenantInfo pagination.TenantInfo
	Basis      *shipmentports.ShipmentQuickFilterBasis
}

type ShipmentDeliveryRow struct {
	ShipmentID    pulid.ID
	ProNumber     string
	StageRank     int16
	CustomerName  string
	City          string
	DeliveryAt    int64
	ActualArrival *int64
	Cutoff        int64
}

type ShipmentPickupRow struct {
	ShipmentID      pulid.ID
	PickupAt        int64
	OriginCity      string
	DestinationCity string
}

type ShipmentDwellRow struct {
	ShipmentID    pulid.ID
	MoveID        pulid.ID
	StopID        pulid.ID
	FacilityName  string
	ActualArrival int64
}

type ShipmentBillingCustomerRow struct {
	CustomerID     pulid.ID
	Name           string
	Count          int
	Total          decimal.Decimal
	TotalCustomers int
}

type ListReadyToBillCustomersRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type ShipmentWatchlistRepository interface {
	ListDeliveriesToday(
		ctx context.Context,
		req *ShipmentWatchlistRequest,
	) ([]*ShipmentDeliveryRow, error)
	NextUncoveredPickup(
		ctx context.Context,
		req *ShipmentWatchlistRequest,
	) (*ShipmentPickupRow, error)
	ListDwellingStops(
		ctx context.Context,
		req *ShipmentWatchlistRequest,
	) ([]*ShipmentDwellRow, error)
	ListReadyToBillCustomers(
		ctx context.Context,
		req *ListReadyToBillCustomersRequest,
	) ([]*ShipmentBillingCustomerRow, error)
}

type ShipmentLateReason struct {
	Label string
	Count int
}

type ShipmentBriefingRepository interface {
	LeadingLateReason(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentLateReason, error)
}

type CarrierPerformance struct {
	CarrierID      pulid.ID
	OffersAnswered int
	OffersAccepted int
	PerMileLoads   int
	AvgRatePerMile decimal.NullDecimal
}

func (p *CarrierPerformance) AcceptancePercent() (float64, bool) {
	if p == nil || p.OffersAnswered == 0 {
		return 0, false
	}

	return float64(p.OffersAccepted) * 100 / float64(p.OffersAnswered), true
}

type ListCarrierPerformanceRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	CarrierIDs []pulid.ID
}

type CarrierPerformanceRepository interface {
	ListCarrierPerformance(
		ctx context.Context,
		req *ListCarrierPerformanceRequest,
	) ([]*CarrierPerformance, error)
}
