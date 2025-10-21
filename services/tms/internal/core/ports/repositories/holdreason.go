package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListHoldReasonRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetHoldReasonByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type HoldReasonRepository interface {
	List(
		ctx context.Context,
		req *ListHoldReasonRequest,
	) (*pagination.ListResult[*holdreason.HoldReason], error)
	GetByID(
		ctx context.Context,
		req GetHoldReasonByIDRequest,
	) (*holdreason.HoldReason, error)
	Create(
		ctx context.Context,
		h *holdreason.HoldReason,
	) (*holdreason.HoldReason, error)
	Update(
		ctx context.Context,
		h *holdreason.HoldReason,
	) (*holdreason.HoldReason, error)
}
