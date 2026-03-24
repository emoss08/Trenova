package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListShipmentTypesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetShipmentTypeByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateShipmentTypeStatusRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	ShipmentTypeIDs []pulid.ID            `json:"shipmentTypeIds"`
	Status          domaintypes.Status    `json:"status"`
}

type GetShipmentTypesByIDsRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	ShipmentTypeIDs []pulid.ID            `json:"shipmentTypeIds"`
}

type ShipmentTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Classes            []string                       `json:"classes"`
}

type ShipmentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListShipmentTypesRequest,
	) (*pagination.ListResult[*shipmenttype.ShipmentType], error)
	GetByID(
		ctx context.Context,
		req GetShipmentTypeByIDRequest,
	) (*shipmenttype.ShipmentType, error)
	GetByIDs(
		ctx context.Context,
		req GetShipmentTypesByIDsRequest,
	) ([]*shipmenttype.ShipmentType, error)
	Create(
		ctx context.Context,
		entity *shipmenttype.ShipmentType,
	) (*shipmenttype.ShipmentType, error)
	Update(
		ctx context.Context,
		entity *shipmenttype.ShipmentType,
	) (*shipmenttype.ShipmentType, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateShipmentTypeStatusRequest,
	) ([]*shipmenttype.ShipmentType, error)
	SelectOptions(
		ctx context.Context,
		req *ShipmentTypeSelectOptionsRequest,
	) (*pagination.ListResult[*shipmenttype.ShipmentType], error)
}
