package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/commodity"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type GetCommodityByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type CommodityRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*commodity.Commodity], error)
	GetByID(ctx context.Context, opts GetCommodityByIDOptions) (*commodity.Commodity, error)
	Create(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
	Update(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
}
