package locationcategoryports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListLocationCategoriesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListLocationCategoryConnectionRequest struct {
	Filter                  *pagination.QueryOptions `json:"filter"`
	Cursor                  pagination.CursorInfo    `json:"-"`
	LocationCategoryColumns []string                 `json:"-"`
}

type GetLocationCategoryByIDRequest struct {
	ID         pulid.ID              `json:"id" form:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetLocationCategoriesByIDsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	LocationCategoryIDs []pulid.ID            `json:"locationCategoryIds"`
}

type LocationCategoryRepository interface {
	List(
		ctx context.Context,
		req *ListLocationCategoriesRequest,
	) (*pagination.ListResult[*locationcategory.LocationCategory], error)
	ListConnection(
		ctx context.Context,
		req *ListLocationCategoryConnectionRequest,
	) (*pagination.CursorListResult[*locationcategory.LocationCategory], error)
	GetByID(
		ctx context.Context,
		req GetLocationCategoryByIDRequest,
	) (*locationcategory.LocationCategory, error)
	GetByIDs(
		ctx context.Context,
		req GetLocationCategoriesByIDsRequest,
	) ([]*locationcategory.LocationCategory, error)
	Create(
		ctx context.Context,
		entity *locationcategory.LocationCategory,
	) (*locationcategory.LocationCategory, error)
	Update(
		ctx context.Context,
		entity *locationcategory.LocationCategory,
	) (*locationcategory.LocationCategory, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*locationcategory.LocationCategory], error)
}
