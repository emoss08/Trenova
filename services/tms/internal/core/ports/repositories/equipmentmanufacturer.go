/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type EquipmentManufacturerFilterOptions struct {
	Status string `query:"status"`
}

type ListEquipmentManufacturerOptions struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions EquipmentManufacturerFilterOptions `query:"filterOptions"`
}

type GetEquipmentManufacturerByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type EquipmentManufacturerRepository interface {
	List(
		ctx context.Context,
		opts ListEquipmentManufacturerOptions,
	) (*ports.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error)
	GetByID(
		ctx context.Context,
		opts GetEquipmentManufacturerByIDOptions,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Create(
		ctx context.Context,
		em *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
	Update(
		ctx context.Context,
		em *equipmentmanufacturer.EquipmentManufacturer,
	) (*equipmentmanufacturer.EquipmentManufacturer, error)
}
