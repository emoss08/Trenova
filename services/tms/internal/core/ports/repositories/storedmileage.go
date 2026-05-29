package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListStoredMileageRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetStoredMileageByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteStoredMileageRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type StoredMileageLookupRequest struct {
	TenantInfo        pagination.TenantInfo
	RouteHash         string
	DistanceUnits     string
	RoutingType       string
	Method            string
	DistanceProfileID pulid.ID
	HazmatSignature   string
}

type StoredMileageRepository interface {
	List(ctx context.Context, req *ListStoredMileageRequest) (*pagination.ListResult[*storedmileage.StoredMileage], error)
	GetByID(ctx context.Context, req GetStoredMileageByIDRequest) (*storedmileage.StoredMileage, error)
	Lookup(ctx context.Context, req StoredMileageLookupRequest) (*storedmileage.StoredMileage, error)
	BulkUpsert(ctx context.Context, entities []*storedmileage.StoredMileage) error
	IncrementHit(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) error
	Deactivate(ctx context.Context, req DeleteStoredMileageRequest) error
}

type StoredMileageBufferRepository interface {
	Push(ctx context.Context, candidate *storedmileage.StoredMileage) error
	PopTenantBatches(ctx context.Context, batchSize int, totalLimit int) ([][]*storedmileage.StoredMileage, error)
	Size(ctx context.Context) (int64, error)
}
