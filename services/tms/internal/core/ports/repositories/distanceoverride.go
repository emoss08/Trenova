package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDistanceOverrideRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetDistanceOverrideByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type DeleteDistanceOverrideRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DistanceOverrideRepository interface {
	List(
		ctx context.Context,
		req *ListDistanceOverrideRequest,
	) (*pagination.ListResult[*distanceoverride.DistanceOverride], error)
	GetByID(
		ctx context.Context,
		req GetDistanceOverrideByIDRequest,
	) (*distanceoverride.DistanceOverride, error)
	Create(
		ctx context.Context,
		entity *distanceoverride.DistanceOverride,
	) (*distanceoverride.DistanceOverride, error)
	Update(
		ctx context.Context,
		entity *distanceoverride.DistanceOverride,
	) (*distanceoverride.DistanceOverride, error)
	Delete(
		ctx context.Context,
		req DeleteDistanceOverrideRequest,
	) error
}
