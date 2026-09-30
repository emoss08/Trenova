package resolver

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

func (r *Resolver) resolveEquipmentTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.equipmentTypeService.Get(
				ctx,
				repositories.GetEquipmentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, equipmentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.equipmentTypeService.SelectOptions(
		ctx,
		&repositories.EquipmentTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			Classes:            equipmentTypeClassesFilter(req.filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		equipmentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveEquipmentManufacturerSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.equipmentManufacturerService.GetByIDs(
			ctx,
			repositories.GetEquipmentManufacturersByIDsRequest{
				TenantInfo:               req.tenantInfo,
				EquipmentManufacturerIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			equipmentManufacturerID,
			equipmentManufacturerSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.equipmentManufacturerService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		equipmentManufacturerSelectOptionItem,
	)
}

func (r *Resolver) resolveTrailerSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.trailerService.GetByIDs(
			ctx,
			repositories.GetTrailersByIDsRequest{
				TenantInfo: req.tenantInfo,
				TrailerIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, trailerID, trailerSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.trailerService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		trailerSelectOptionItem,
	)
}

func (r *Resolver) resolveTractorSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.tractorService.GetByIDs(
			ctx,
			repositories.GetTractorsByIDsRequest{
				TenantInfo: req.tenantInfo,
				TractorIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, tractorID, tractorSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.tractorService.SelectOptions(
		ctx,
		&repositories.TractorSelectOptionsRequest{
			SelectOptionsRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
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
