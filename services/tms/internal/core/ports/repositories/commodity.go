package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCommodityRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetCommodityByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateCommodityStatusRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	CommodityIDs []pulid.ID            `json:"commodityIds"`
	Status       domaintypes.Status    `json:"status"`
}

type GetCommoditiesByIDsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	CommodityIDs []pulid.ID            `json:"commodityIds"`
}

type CommoditySelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type CommodityRepository interface {
	List(
		ctx context.Context,
		req *ListCommodityRequest,
	) (*pagination.ListResult[*commodity.Commodity], error)
	GetByID(
		ctx context.Context,
		req GetCommodityByIDRequest,
	) (*commodity.Commodity, error)
	GetByIDs(
		ctx context.Context,
		req GetCommoditiesByIDsRequest,
	) ([]*commodity.Commodity, error)
	Create(
		ctx context.Context,
		entity *commodity.Commodity,
	) (*commodity.Commodity, error)
	Update(
		ctx context.Context,
		entity *commodity.Commodity,
	) (*commodity.Commodity, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateCommodityStatusRequest,
	) ([]*commodity.Commodity, error)
	SelectOptions(
		ctx context.Context,
		req *CommoditySelectOptionsRequest,
	) (*pagination.ListResult[*commodity.Commodity], error)
}
