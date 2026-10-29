package shipmentboardports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/boardgrouping"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type ShipmentBoardCapabilities struct {
	AI            bool
	OperationType tenant.OperationType
	HOS           bool
	Maps          bool
}

type ShipmentBoardCapabilitiesReader interface {
	Capabilities(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentBoardCapabilities, error)
}

type ShipmentBoardScopeRequest struct {
	Filter       *pagination.QueryOptions
	QuickFilters []quickfilter.QuickFilterSpec
	Timezone     string
	// Margins is whether the reader holds the margins field group. Without it
	// a filter that selects loads by margin is refused and not counted.
	Margins bool
}

type ShipmentStageSummary struct {
	Stage   shipmentvocab.Stage
	Rank    int
	Count   int
	Revenue decimal.Decimal
}

type ShipmentStageSummaryReader interface {
	StageSummary(
		ctx context.Context,
		req *ShipmentBoardScopeRequest,
	) ([]*ShipmentStageSummary, error)
}

type ShipmentBoardGroup struct {
	Key     string
	Label   string
	Count   int
	Revenue decimal.Decimal
}

type SummarizeShipmentBoardGroupsRequest struct {
	Scope   *ShipmentBoardScopeRequest
	GroupBy boardgrouping.BoardGrouping
}

type ShipmentBoardGroupSummarizer interface {
	GroupSummary(
		ctx context.Context,
		req *SummarizeShipmentBoardGroupsRequest,
	) ([]*ShipmentBoardGroup, error)
}

type ShipmentQuickFilterCount struct {
	Filter quickfilter.QuickFilter
	Count  int
}

type ShipmentQuickFilterCounter interface {
	QuickFilterCounts(
		ctx context.Context,
		req *ShipmentBoardScopeRequest,
	) ([]*ShipmentQuickFilterCount, error)
}
