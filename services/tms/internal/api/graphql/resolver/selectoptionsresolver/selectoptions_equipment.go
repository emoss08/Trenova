package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveEquipmentTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EquipmentTypeService.Get(
				ctx,
				repositories.GetEquipmentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, equipmentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EquipmentTypeService.SelectOptions(
		ctx,
		&repositories.EquipmentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Classes:            equipmentTypeClassesFilter(req.Filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		equipmentTypeSelectOptionItem,
	)
}

func (r *Deps) resolveEquipmentManufacturerSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EquipmentManufacturerService.GetByIDs(
			ctx,
			repositories.GetEquipmentManufacturersByIDsRequest{
				TenantInfo:               req.TenantInfo,
				EquipmentManufacturerIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.IDs,
			entities,
			equipmentManufacturerID,
			equipmentManufacturerSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.EquipmentManufacturerService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		equipmentManufacturerSelectOptionItem,
	)
}

func (r *Deps) resolveTrailerSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.TrailerService.GetByIDs(
			ctx,
			repositories.GetTrailersByIDsRequest{
				TenantInfo: req.TenantInfo,
				TrailerIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.IDs, entities, trailerID, trailerSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.TrailerService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		trailerSelectOptionItem,
	)
}

func (r *Deps) resolveTractorSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.TractorService.GetByIDs(
			ctx,
			repositories.GetTractorsByIDsRequest{
				TenantInfo: req.TenantInfo,
				TractorIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.IDs, entities, tractorID, tractorSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.TractorService.SelectOptions(
		ctx,
		&repositories.TractorSelectOptionsRequest{
			SelectOptionsRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		tractorSelectOptionItem,
	)
}

func equipmentTypeSelectOptionItem(entity *equipmenttype.EquipmentType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		equipmentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func equipmentManufacturerSelectOptionItem(
	entity *equipmentmanufacturer.EquipmentManufacturer,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		equipmentManufacturerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func equipmentTypeSelectOption(entity *equipmenttype.EquipmentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"color": entity.Color,
			"class": entity.Class,
		},
	}
}

func equipmentManufacturerSelectOption(
	entity *equipmentmanufacturer.EquipmentManufacturer,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
	}
}

func trailerSelectOptionItem(entity *trailer.Trailer) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		trailerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func trailerSelectOption(entity *trailer.Trailer) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: entity.Code,
	}
}

func tractorSelectOptionItem(entity *tractor.Tractor) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		tractorSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func tractorSelectOption(entity *tractor.Tractor) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: entity.Code,
		Meta: map[string]any{
			"primaryWorkerId":   optionalIDMeta(entity.PrimaryWorkerID),
			"secondaryWorkerId": optionalIDMeta(entity.SecondaryWorkerID),
		},
	}
}

func trailerID(entity *trailer.Trailer) pulid.ID {
	return entity.ID
}

func tractorID(entity *tractor.Tractor) pulid.ID {
	return entity.ID
}

func equipmentManufacturerID(entity *equipmentmanufacturer.EquipmentManufacturer) pulid.ID {
	return entity.ID
}
