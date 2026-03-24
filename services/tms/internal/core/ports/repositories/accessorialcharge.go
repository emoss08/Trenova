package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAccessorialChargeRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetAccessorialChargeByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type AccessorialChargeRepository interface {
	List(
		ctx context.Context,
		req *ListAccessorialChargeRequest,
	) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error)
	GetByID(
		ctx context.Context,
		req GetAccessorialChargeByIDRequest,
	) (*accessorialcharge.AccessorialCharge, error)
	Create(
		ctx context.Context,
		entity *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
	Update(
		ctx context.Context,
		entity *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
}
