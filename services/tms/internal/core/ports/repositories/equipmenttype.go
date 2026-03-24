package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEquipmentTypesRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Classes []string                 `json:"classes"`
}

type GetEquipmentTypeByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type BulkUpdateEquipmentTypeStatusRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	EquipmentTypeIDs []pulid.ID            `json:"equipmentTypeIds"`
	Status           domaintypes.Status    `json:"status"`
}

type GetEquipmentTypesByIDsRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	EquipmentTypeIDs []pulid.ID            `json:"equipmentTypeIds"`
}

type EquipmentTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	Classes            []string                       `json:"classes"`
}

type EquipmentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListEquipmentTypesRequest,
	) (*pagination.ListResult[*equipmenttype.EquipmentType], error)
	GetByID(
		ctx context.Context,
		req GetEquipmentTypeByIDRequest,
	) (*equipmenttype.EquipmentType, error)
	GetByIDs(
		ctx context.Context,
		req GetEquipmentTypesByIDsRequest,
	) ([]*equipmenttype.EquipmentType, error)
	Create(
		ctx context.Context,
		entity *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
	Update(
		ctx context.Context,
		entity *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateEquipmentTypeStatusRequest,
	) ([]*equipmenttype.EquipmentType, error)
	SelectOptions(
		ctx context.Context,
		req *EquipmentTypeSelectOptionsRequest,
	) (*pagination.ListResult[*equipmenttype.EquipmentType], error)
}
