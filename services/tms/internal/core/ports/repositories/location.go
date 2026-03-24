package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListLocationRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetLocationByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateLocationStatusRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	LocationIDs []pulid.ID            `json:"locationIds"`
	Status      domaintypes.Status    `json:"status"`
}

type GetLocationsByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	LocationIDs []pulid.ID            `json:"locationIds"`
}

type LocationSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type LocationRepository interface {
	List(
		ctx context.Context,
		req *ListLocationRequest,
	) (*pagination.ListResult[*location.Location], error)
	GetByID(
		ctx context.Context,
		req GetLocationByIDRequest,
	) (*location.Location, error)
	GetByIDs(
		ctx context.Context,
		req GetLocationsByIDsRequest,
	) ([]*location.Location, error)
	Create(
		ctx context.Context,
		entity *location.Location,
	) (*location.Location, error)
	Update(
		ctx context.Context,
		entity *location.Location,
	) (*location.Location, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateLocationStatusRequest,
	) ([]*location.Location, error)
	SelectOptions(
		ctx context.Context,
		req *LocationSelectOptionsRequest,
	) (*pagination.ListResult[*location.Location], error)
}
