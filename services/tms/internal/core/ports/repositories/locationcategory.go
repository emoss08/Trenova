package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetLocationCategoryByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type ListLocationCategoryRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type LocationCategoryRepository interface {
	List(
		ctx context.Context,
		req *ListLocationCategoryRequest,
	) (*pagination.ListResult[*location.LocationCategory], error)
	GetByID(
		ctx context.Context,
		req GetLocationCategoryByIDRequest,
	) (*location.LocationCategory, error)
	Create(ctx context.Context, lc *location.LocationCategory) (*location.LocationCategory, error)
	Update(ctx context.Context, lc *location.LocationCategory) (*location.LocationCategory, error)
}
