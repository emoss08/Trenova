package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetAccessorialChargeByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type ListAccessorialChargeRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type AccessorialChargeRepository interface {
	List(
		ctx context.Context,
		req *ListAccessorialChargeRequest,
	) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error)
	GetByID(
		ctx context.Context,
		req GetAccessorialChargeByIDRequest,
	) (*accessorialcharge.AccessorialCharge, error)
	Create(
		ctx context.Context,
		a *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
	Update(
		ctx context.Context,
		a *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
}
