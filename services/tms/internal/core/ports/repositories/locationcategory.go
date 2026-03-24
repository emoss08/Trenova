package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListLocationCategoriesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetLocationCategoryByIDRequest struct {
	ID         pulid.ID              `json:"id" form:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type LocationCategoryRepository interface {
	List(
		ctx context.Context,
		req *ListLocationCategoriesRequest,
	) (*pagination.ListResult[*locationcategory.LocationCategory], error)
	GetByID(
		ctx context.Context,
		req GetLocationCategoryByIDRequest,
	) (*locationcategory.LocationCategory, error)
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
