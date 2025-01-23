package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetEquipmentTypeByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type EquipmentTypeRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*equipmenttype.EquipmentType], error)
	GetByID(ctx context.Context, opts GetEquipmentTypeByIDOptions) (*equipmenttype.EquipmentType, error)
	Create(ctx context.Context, et *equipmenttype.EquipmentType) (*equipmenttype.EquipmentType, error)
	Update(ctx context.Context, et *equipmenttype.EquipmentType) (*equipmenttype.EquipmentType, error)
}
