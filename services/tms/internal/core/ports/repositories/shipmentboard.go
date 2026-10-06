package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ShipmentFacet string

const (
	ShipmentFacetStatus       = ShipmentFacet("Status")
	ShipmentFacetEquipment    = ShipmentFacet("Equipment")
	ShipmentFacetTenderStatus = ShipmentFacet("TenderStatus")
	ShipmentFacetCustomer     = ShipmentFacet("Customer")
)

func (f ShipmentFacet) IsValid() bool {
	switch f {
	case ShipmentFacetStatus,
		ShipmentFacetEquipment,
		ShipmentFacetTenderStatus,
		ShipmentFacetCustomer:
		return true
	default:
		return false
	}
}

type ShipmentBoardScope struct {
	Filter  *pagination.QueryOptions
	Options ShipmentOptions
}

type ShipmentStageSummaryRow struct {
	StageRank int16           `bun:"stage_rank"`
	Count     int             `bun:"count"`
	Revenue   decimal.Decimal `bun:"revenue"`
}

type ShipmentFacetValue struct {
	Value string `bun:"value"`
	Label string `bun:"label"`
	Count int    `bun:"count"`
}

type ShipmentFacetCounts struct {
	Facet  ShipmentFacet
	Field  string
	Values []*ShipmentFacetValue
}

type ShipmentQuickFilterTotal struct {
	Count   int
	Revenue decimal.Decimal
}

type CountShipmentQuickFiltersRequest struct {
	Scope   *ShipmentBoardScope
	Filters []shipment.QuickFilterSpec
}

type CountShipmentFacetRequest struct {
	Scope *ShipmentBoardScope
	Facet ShipmentFacet
	Limit int
}

type ShipmentBoardRepository interface {
	StageSummary(ctx context.Context, scope *ShipmentBoardScope) ([]*ShipmentStageSummaryRow, error)
	QuickFilterTotals(
		ctx context.Context,
		req *CountShipmentQuickFiltersRequest,
	) ([]ShipmentQuickFilterTotal, error)
	FacetCounts(ctx context.Context, req *CountShipmentFacetRequest) (*ShipmentFacetCounts, error)
}

type ShipmentWatchlistRequest struct {
	TenantInfo pagination.TenantInfo
	Basis      *ShipmentQuickFilterBasis
}

type ShipmentDeliveryRow struct {
	ShipmentID    pulid.ID `bun:"shipment_id"`
	ProNumber     string   `bun:"pro_number"`
	StageRank     int16    `bun:"stage_rank"`
	CustomerName  string   `bun:"customer_name"`
	City          string   `bun:"city"`
	DeliveryAt    int64    `bun:"delivery_at"`
	ActualArrival *int64   `bun:"actual_arrival"`
	Cutoff        int64    `bun:"cutoff"`
}

type ShipmentPickupRow struct {
	ShipmentID      pulid.ID `bun:"shipment_id"`
	PickupAt        int64    `bun:"pickup_at"`
	OriginCity      string   `bun:"origin_city"`
	DestinationCity string   `bun:"destination_city"`
}

type ShipmentDwellRow struct {
	ShipmentID    pulid.ID `bun:"shipment_id"`
	MoveID        pulid.ID `bun:"move_id"`
	StopID        pulid.ID `bun:"stop_id"`
	FacilityName  string   `bun:"facility_name"`
	ActualArrival int64    `bun:"actual_arrival"`
}

type ShipmentBillingCustomerRow struct {
	CustomerID     pulid.ID        `bun:"customer_id"`
	Name           string          `bun:"name"`
	Count          int             `bun:"count"`
	Total          decimal.Decimal `bun:"total"`
	TotalCustomers int             `bun:"total_customers"`
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
	Label string `bun:"label"`
	Count int    `bun:"count"`
}

type ShipmentBriefingRepository interface {
	LeadingLateReason(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentLateReason, error)
}
