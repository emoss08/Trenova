package accessorialchargeports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAccessorialChargeRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListAccessorialChargeConnectionRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	Cursor                   pagination.CursorInfo    `json:"-"`
	AccessorialChargeColumns []string                 `json:"-"`
}

type GetAccessorialChargeByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type GetAccessorialChargesByIDsRequest struct {
	TenantInfo           pagination.TenantInfo `json:"-"`
	AccessorialChargeIDs []pulid.ID            `json:"accessorialChargeIds"`
}

type AccessorialChargeRepository interface {
	List(
		ctx context.Context,
		req *ListAccessorialChargeRequest,
	) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error)
	ListConnection(
		ctx context.Context,
		req *ListAccessorialChargeConnectionRequest,
	) (*pagination.CursorListResult[*accessorialcharge.AccessorialCharge], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error)
	GetByID(
		ctx context.Context,
		req GetAccessorialChargeByIDRequest,
	) (*accessorialcharge.AccessorialCharge, error)
	GetByIDs(
		ctx context.Context,
		req GetAccessorialChargesByIDsRequest,
	) ([]*accessorialcharge.AccessorialCharge, error)
	ListCatalog(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*accessorialcharge.AccessorialCharge, error)
	Create(
		ctx context.Context,
		entity *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
	Update(
		ctx context.Context,
		entity *accessorialcharge.AccessorialCharge,
	) (*accessorialcharge.AccessorialCharge, error)
}
