/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFavoritesOptions struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type GetFavoriteByIDOptions struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	FavoriteID pulid.ID
}

type DeleteFavoriteOptions struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	FavoriteID pulid.ID
}

type GetFavoriteByURLOptions struct {
	OrgID   pulid.ID
	BuID    pulid.ID
	UserID  pulid.ID
	PageURL string
}

type FavoriteRepository interface {
	List(ctx context.Context, opts ListFavoritesOptions) ([]*pagefavorite.PageFavorite, error)
	GetByID(ctx context.Context, opts GetFavoriteByIDOptions) (*pagefavorite.PageFavorite, error)
	GetByURL(ctx context.Context, opts GetFavoriteByURLOptions) (*pagefavorite.PageFavorite, error)
	Create(ctx context.Context, f *pagefavorite.PageFavorite) (*pagefavorite.PageFavorite, error)
	Update(ctx context.Context, f *pagefavorite.PageFavorite) (*pagefavorite.PageFavorite, error)
	Delete(ctx context.Context, opts DeleteFavoriteOptions) error
}
