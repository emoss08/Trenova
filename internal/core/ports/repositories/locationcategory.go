/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetLocationCategoryByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type LocationCategoryRepository interface {
	List(
		ctx context.Context,
		opts *ports.LimitOffsetQueryOptions,
	) (*ports.ListResult[*location.LocationCategory], error)
	GetByID(
		ctx context.Context,
		opts GetLocationCategoryByIDOptions,
	) (*location.LocationCategory, error)
	Create(ctx context.Context, lc *location.LocationCategory) (*location.LocationCategory, error)
	Update(ctx context.Context, lc *location.LocationCategory) (*location.LocationCategory, error)
}
