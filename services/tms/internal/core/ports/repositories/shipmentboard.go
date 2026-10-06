package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
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
	QuickFilterCounts(
		ctx context.Context,
		req *CountShipmentQuickFiltersRequest,
	) (map[shipment.QuickFilter]int, error)
	FacetCounts(ctx context.Context, req *CountShipmentFacetRequest) (*ShipmentFacetCounts, error)
}
