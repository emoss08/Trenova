package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetEquipManufacturerByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type EquipmentManufacturerRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error)
	GetByID(ctx context.Context, opts GetEquipManufacturerByIDOptions) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Create(ctx context.Context, em *equipmentmanufacturer.EquipmentManufacturer) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Update(ctx context.Context, em *equipmentmanufacturer.EquipmentManufacturer) (*equipmentmanufacturer.EquipmentManufacturer, error)
}
