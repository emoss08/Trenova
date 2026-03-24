package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEquipmentManufacturersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEquipmentManufacturerByIDRequest struct {
	ID         pulid.ID              `json:"id" form:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type BulkUpdateEquipmentManufacturerStatusRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	EquipmentManufacturerIDs []pulid.ID            `json:"equipmentManufacturerIds"`
	Status                   domaintypes.Status    `json:"status"`
}

type GetEquipmentManufacturersByIDsRequest struct {
	TenantInfo               pagination.TenantInfo `json:"-"`
	EquipmentManufacturerIDs []pulid.ID            `json:"equipmentManufacturerIds"`
}

type EquipmentManufacturerRepository interface {
	List(
		ctx context.Context,
		req *ListEquipmentManufacturersRequest,
	) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error)
	GetByID(
		ctx context.Context,
		req GetEquipmentManufacturerByIDRequest,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	GetByIDs(
		ctx context.Context,
		req GetEquipmentManufacturersByIDsRequest,
	) ([]*equipmentmanufacturer.EquipmentManufacturer, error)
	Create(
		ctx context.Context,
		entity *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Update(
		ctx context.Context,
		entity *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateEquipmentManufacturerStatusRequest,
	) ([]*equipmentmanufacturer.EquipmentManufacturer, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error)
}
