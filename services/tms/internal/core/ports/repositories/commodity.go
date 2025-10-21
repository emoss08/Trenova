package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListCommodityRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetCommodityByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type CommodityRepository interface {
	List(
		ctx context.Context,
		req *ListCommodityRequest,
	) (*pagination.ListResult[*commodity.Commodity], error)
	GetByID(ctx context.Context, opts GetCommodityByIDRequest) (*commodity.Commodity, error)
	Create(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
	Update(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
}
