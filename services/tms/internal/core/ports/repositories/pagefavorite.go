package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pagefavorite"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListPageFavoritesRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetPageFavoriteByURLRequest struct {
	PageURL    string
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type DeletePageFavoriteRequest struct {
	FavoriteID pulid.ID
	TenantInfo pagination.TenantInfo
}

type PageFavoriteRepository interface {
	List(
		ctx context.Context,
		req *ListPageFavoritesRequest,
	) ([]*pagefavorite.PageFavorite, error)
	GetByURL(
		ctx context.Context,
		req *GetPageFavoriteByURLRequest,
	) (*pagefavorite.PageFavorite, bool, error)
	Create(
		ctx context.Context,
		entity *pagefavorite.PageFavorite,
	) (*pagefavorite.PageFavorite, error)
	Delete(
		ctx context.Context,
		id pulid.ID,
		tenantInfo pagination.TenantInfo,
	) error
}
