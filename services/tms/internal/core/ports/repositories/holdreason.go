package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListHoldReasonRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListHoldReasonConnectionRequest struct {
	Filter            *pagination.QueryOptions `json:"filter"`
	Cursor            pagination.CursorInfo    `json:"-"`
	HoldReasonColumns []string                 `json:"-"`
}

type GetHoldReasonByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type HoldReasonSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type HoldReasonRepository interface {
	List(
		ctx context.Context,
		req *ListHoldReasonRequest,
	) (*pagination.ListResult[*holdreason.HoldReason], error)
	ListConnection(
		ctx context.Context,
		req *ListHoldReasonConnectionRequest,
	) (*pagination.CursorListResult[*holdreason.HoldReason], error)
	GetByID(
		ctx context.Context,
		req GetHoldReasonByIDRequest,
	) (*holdreason.HoldReason, error)
	Create(
		ctx context.Context,
		entity *holdreason.HoldReason,
	) (*holdreason.HoldReason, error)
	Update(
		ctx context.Context,
		entity *holdreason.HoldReason,
	) (*holdreason.HoldReason, error)
	SelectOptions(
		ctx context.Context,
		req *HoldReasonSelectOptionsRequest,
	) (*pagination.ListResult[*holdreason.HoldReason], error)
}
