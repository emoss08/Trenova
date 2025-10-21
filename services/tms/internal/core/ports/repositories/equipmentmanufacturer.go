package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListEquipmentManufacturerRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetEquipmentManufacturerByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type EquipmentManufacturerRepository interface {
	List(
		ctx context.Context,
		req *ListEquipmentManufacturerRequest,
	) (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error)
	GetByID(
		ctx context.Context,
		req GetEquipmentManufacturerByIDRequest,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Create(
		ctx context.Context,
		entity *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Update(
		ctx context.Context,
		entity *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
}
