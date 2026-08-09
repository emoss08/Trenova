package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CarrierFilterOptions struct {
	IncludeState             bool `form:"includeState"`
	IncludeContacts          bool `form:"includeContacts"`
	IncludeInsurancePolicies bool `form:"includeInsurancePolicies"`
}

type ListCarrierRequest struct {
	Filter               *pagination.QueryOptions `json:"filter" form:"filter"`
	CarrierFilterOptions ` form:"carrierFilterOptions"`
}

type ListCarrierConnectionRequest struct {
	Filter               *pagination.QueryOptions `json:"filter"`
	Cursor               pagination.CursorInfo    `json:"-"`
	CarrierColumns       []string                 `json:"-"`
	CarrierFilterOptions `json:"-"`
}

type GetCarrierByIDRequest struct {
	ID                   pulid.ID              `json:"id"         form:"id"`
	TenantInfo           pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	CarrierFilterOptions ` form:"carrierFilterOptions"`
}

type BulkUpdateCarrierStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CarrierIDs []pulid.ID            `json:"carrierIds"`
	Status     carrier.Status        `json:"status"`
}

type GetCarriersByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CarrierIDs []pulid.ID            `json:"carrierIds"`
}

type CarrierSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type CarrierRepository interface {
	List(
		ctx context.Context,
		req *ListCarrierRequest,
	) (*pagination.ListResult[*carrier.Carrier], error)
	ListConnection(
		ctx context.Context,
		req *ListCarrierConnectionRequest,
	) (*pagination.CursorListResult[*carrier.Carrier], error)
	GetByID(
		ctx context.Context,
		req GetCarrierByIDRequest,
	) (*carrier.Carrier, error)
	GetByIDs(
		ctx context.Context,
		req GetCarriersByIDsRequest,
	) ([]*carrier.Carrier, error)
	Create(
		ctx context.Context,
		entity *carrier.Carrier,
	) (*carrier.Carrier, error)
	Update(
		ctx context.Context,
		entity *carrier.Carrier,
	) (*carrier.Carrier, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateCarrierStatusRequest,
	) ([]*carrier.Carrier, error)
	SelectOptions(
		ctx context.Context,
		req *CarrierSelectOptionsRequest,
	) (*pagination.ListResult[*carrier.Carrier], error)
}
