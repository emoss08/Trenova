package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListCarrierCapacityPostingsRequest struct {
	Filter    *pagination.QueryOptions
	Cursor    pagination.CursorInfo
	CarrierID pulid.ID
	OpenAt    int64
}

type GetCarrierCapacityPostingRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type DeleteCarrierCapacityPostingRequest struct {
	ID         pulid.ID
	Version    int64
	TenantInfo pagination.TenantInfo
}

type ListOpenCarrierCapacityRequest struct {
	TenantInfo pagination.TenantInfo
	OpenAt     int64
	Through    int64
	CarrierIDs []pulid.ID
	Limit      int
}

type CarrierCapacityPostingRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierCapacityPostingsRequest,
	) (*pagination.ListResult[*carriercapacity.Posting], error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierCapacityPostingsRequest,
	) (*pagination.CursorListResult[*carriercapacity.Posting], error)
	GetByID(
		ctx context.Context,
		req *GetCarrierCapacityPostingRequest,
	) (*carriercapacity.Posting, error)
	Create(ctx context.Context, entity *carriercapacity.Posting) (*carriercapacity.Posting, error)
	Update(ctx context.Context, entity *carriercapacity.Posting) (*carriercapacity.Posting, error)
	Delete(ctx context.Context, req *DeleteCarrierCapacityPostingRequest) error
	ListOpen(
		ctx context.Context,
		req *ListOpenCarrierCapacityRequest,
	) ([]*carriercapacity.Posting, error)
}
