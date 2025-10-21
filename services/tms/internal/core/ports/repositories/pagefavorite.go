package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetPageFavoriteByIDRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	FavoriteID pulid.ID
}

type CreatePageFavoriteRequest struct {
	OrgID    pulid.ID
	BuID     pulid.ID
	UserID   pulid.ID
	Favorite *pagefavorite.PageFavorite
}

type UpdatePageFavoriteRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	FavoriteID pulid.ID
	Favorite   *pagefavorite.PageFavorite
}

type DeletePageFavoriteRequest struct {
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
	FavoriteID pulid.ID
}

type GetPageFavoriteByURLRequest struct {
	OrgID   pulid.ID
	BuID    pulid.ID
	UserID  pulid.ID
	PageURL string
}

type PageFavoriteRepository interface {
	List(
		ctx context.Context,
		opts *pagination.QueryOptions,
	) (*pagination.ListResult[*pagefavorite.PageFavorite], error)
	GetByID(
		ctx context.Context,
		req GetPageFavoriteByIDRequest,
	) (*pagefavorite.PageFavorite, error)
	GetByURL(
		ctx context.Context,
		req GetPageFavoriteByURLRequest,
	) (*pagefavorite.PageFavorite, error)
	Create(ctx context.Context, req *CreatePageFavoriteRequest) (*pagefavorite.PageFavorite, error)
	Delete(ctx context.Context, req DeletePageFavoriteRequest) error
}
