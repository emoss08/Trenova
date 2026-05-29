package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDistanceProfileRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetDistanceProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteDistanceProfileRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DistanceProfileRepository interface {
	List(ctx context.Context, req *ListDistanceProfileRequest) (*pagination.ListResult[*distanceprofile.DistanceProfile], error)
	GetByID(ctx context.Context, req GetDistanceProfileByIDRequest) (*distanceprofile.DistanceProfile, error)
	GetDefault(ctx context.Context, tenantInfo pagination.TenantInfo) (*distanceprofile.DistanceProfile, error)
	EnsureDefault(ctx context.Context, tenantInfo pagination.TenantInfo) (*distanceprofile.DistanceProfile, error)
	Create(ctx context.Context, entity *distanceprofile.DistanceProfile) (*distanceprofile.DistanceProfile, error)
	Update(ctx context.Context, entity *distanceprofile.DistanceProfile) (*distanceprofile.DistanceProfile, error)
	Delete(ctx context.Context, req DeleteDistanceProfileRequest) error
	SetDefault(ctx context.Context, req GetDistanceProfileByIDRequest) (*distanceprofile.DistanceProfile, error)
}
