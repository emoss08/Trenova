package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type GetCostingControlRequest struct {
	TenantInfo pagination.TenantInfo
}

type ReplaceCategoryGLAccountsRequest struct {
	TenantInfo     pagination.TenantInfo
	CostCategoryID pulid.ID
	GLAccountIDs   []pulid.ID
}

type CostingControlRepository interface {
	GetByOrgID(
		ctx context.Context,
		req *GetCostingControlRequest,
	) (*costingcontrol.CostingControl, error)

	Update(
		ctx context.Context,
		entity *costingcontrol.CostingControl,
	) (*costingcontrol.CostingControl, error)

	UpdateCategory(
		ctx context.Context,
		category *costingcontrol.CostCategory,
	) (*costingcontrol.CostCategory, error)

	ReplaceCategoryGLAccounts(
		ctx context.Context,
		req *ReplaceCategoryGLAccountsRequest,
	) error
}

type SumExpenseByAccountsRequest struct {
	TenantInfo   pagination.TenantInfo
	GLAccountIDs []pulid.ID
	FromDate     int64
	ToDate       int64
}

type FleetMilesRequest struct {
	TenantInfo pagination.TenantInfo
	FromDate   int64
	ToDate     int64
}

type FleetMilesResult struct {
	LoadedMiles   float64
	DeadheadMiles float64
	TotalMiles    float64
}

type FleetCostAggregatesRequest struct {
	TenantInfo           pagination.TenantInfo
	FromDate             int64
	ToDate               int64
	CostPerMile          decimal.Decimal
	IncludeDeadheadMiles bool
}

type FleetCostAggregatesResult struct {
	ShipmentCount     int
	UnprofitableCount int
	TotalRevenue      decimal.Decimal
	TotalMiles        float64
	LoadedMiles       float64
	DeadheadMiles     float64
}

type ShipmentMilesByIDsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
}

type ShipmentMilesRow struct {
	ShipmentID      pulid.ID        `bun:"shipment_id"`
	LoadedMiles     float64         `bun:"loaded_miles"`
	DeadheadMiles   float64         `bun:"deadhead_miles"`
	Revenue         decimal.Decimal `bun:"revenue"`
	MissingDistance bool            `bun:"missing_distance"`
}

type CostingActualsRepository interface {
	ShipmentMilesByIDs(
		ctx context.Context,
		req *ShipmentMilesByIDsRequest,
	) ([]*ShipmentMilesRow, error)

	SumExpenseByAccounts(
		ctx context.Context,
		req *SumExpenseByAccountsRequest,
	) (map[pulid.ID]decimal.Decimal, error)

	FleetMiles(
		ctx context.Context,
		req *FleetMilesRequest,
	) (*FleetMilesResult, error)

	FleetCostAggregates(
		ctx context.Context,
		req *FleetCostAggregatesRequest,
	) (*FleetCostAggregatesResult, error)
}
