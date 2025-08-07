/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCommodityByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type CommodityRepository interface {
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*commodity.Commodity], error)
	GetByID(ctx context.Context, opts GetCommodityByIDOptions) (*commodity.Commodity, error)
	Create(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
	Update(ctx context.Context, c *commodity.Commodity) (*commodity.Commodity, error)
}
