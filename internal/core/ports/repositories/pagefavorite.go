package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
	List(ctx context.Context) ([]*pagefavorite.PageFavorite, error)
	GetByID(ctx context.Context, opts GetFavoriteByIDOptions) (*pagefavorite.PageFavorite, error)
	GetByURL(ctx context.Context, opts GetFavoriteByURLOptions) (*pagefavorite.PageFavorite, error)
	Create(ctx context.Context, f *pagefavorite.PageFavorite) (*pagefavorite.PageFavorite, error)
	Update(ctx context.Context, f *pagefavorite.PageFavorite) (*pagefavorite.PageFavorite, error)
	Delete(ctx context.Context, opts DeleteFavoriteOptions) error
}
