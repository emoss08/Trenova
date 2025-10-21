package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListEquipmentTypeRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"  form:"filter"`
	Classes []string                 `json:"classes" form:"classes"`
}

type GetEquipmentTypeByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type EquipmentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListEquipmentTypeRequest,
	) (*pagination.ListResult[*equipmenttype.EquipmentType], error)
	GetByID(
		ctx context.Context,
		req GetEquipmentTypeByIDRequest,
	) (*equipmenttype.EquipmentType, error)
	Create(
		ctx context.Context,
		entity *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
	Update(
		ctx context.Context,
		entity *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
}
