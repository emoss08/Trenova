/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListEquipmentTypeRequest struct {
	Filter  *ports.LimitOffsetQueryOptions
	Classes []string `query:"classes"`
}

type GetEquipmentTypeByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type EquipmentTypeRepository interface {
	List(
		ctx context.Context,
		opts *ListEquipmentTypeRequest,
	) (*ports.ListResult[*equipmenttype.EquipmentType], error)
	GetByID(
		ctx context.Context,
		opts GetEquipmentTypeByIDOptions,
	) (*equipmenttype.EquipmentType, error)
	Create(
		ctx context.Context,
		et *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
	Update(
		ctx context.Context,
		et *equipmenttype.EquipmentType,
	) (*equipmenttype.EquipmentType, error)
}
