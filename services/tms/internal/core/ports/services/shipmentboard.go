package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	QuickFilters []shipment.QuickFilterSpec
	Timezone     string
}

type ShipmentStageSummary struct {
	Stage   shipment.Stage
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

type ShipmentQuickFilterCount struct {
	Filter shipment.QuickFilter
	Count  int
}

type ShipmentQuickFilterCounter interface {
	QuickFilterCounts(
		ctx context.Context,
		req *ShipmentBoardScopeRequest,
	) ([]*ShipmentQuickFilterCount, error)
}
