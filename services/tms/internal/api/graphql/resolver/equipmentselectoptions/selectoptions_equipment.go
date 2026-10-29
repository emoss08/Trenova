package equipmentselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/fleetcodeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	EquipmentTypeService         *equipmenttypeservice.Service
	EquipmentManufacturerService *equipmentmanufacturerservice.Service
	TrailerService               *trailerservice.Service
	TractorService               *tractorservice.Service
	FleetCodeService             *fleetcodeservice.Service
}

func (r *Sources) EquipmentType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EquipmentTypeService.GetByIDs(
			ctx,
			equipmenttypeports.GetEquipmentTypesByIDsRequest{
				TenantInfo:       req.TenantInfo,
				EquipmentTypeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*equipmenttype.EquipmentType).GetID,
			equipmentTypeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EquipmentTypeService.SelectOptions(
		ctx,
		&equipmenttypeports.EquipmentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			Classes:            equipmentTypeClassesFilter(req.Filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		equipmentTypeSelectOptionItem,
	)
}

func (r *Sources) EquipmentManufacturer(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.EquipmentManufacturerService.GetByIDs(
			ctx,
			equipmentmanufacturerports.GetEquipmentManufacturersByIDsRequest{
				TenantInfo:               req.TenantInfo,
				EquipmentManufacturerIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			equipmentManufacturerID,
			equipmentManufacturerSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EquipmentManufacturerService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		equipmentManufacturerSelectOptionItem,
	)
}

func (r *Sources) Trailer(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.TrailerService.GetByIDs(
			ctx,
			trailerports.GetTrailersByIDsRequest{
				TenantInfo: req.TenantInfo,
				TrailerIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(req.IDs, entities, trailerID, trailerSelectOptionItem)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.TrailerService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		trailerSelectOptionItem,
	)
}

func (r *Sources) Tractor(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.TractorService.GetByIDs(
			ctx,
			tractorports.GetTractorsByIDsRequest{
				TenantInfo: req.TenantInfo,
				TractorIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(req.IDs, entities, tractorID, tractorSelectOptionItem)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.TractorService.SelectOptions(
		ctx,
		&tractorports.TractorSelectOptionsRequest{
			SelectOptionsRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		tractorSelectOptionItem,
	)
}

func equipmentTypeSelectOptionItem(entity *equipmenttype.EquipmentType) selectoption.Item {
	return selectoption.ItemFor(
		equipmentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func equipmentManufacturerSelectOptionItem(
	entity *equipmentmanufacturer.EquipmentManufacturer,
) selectoption.Item {
	return selectoption.ItemFor(
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

func trailerSelectOptionItem(entity *trailer.Trailer) selectoption.Item {
	return selectoption.ItemFor(
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

func tractorSelectOptionItem(entity *tractor.Tractor) selectoption.Item {
	return selectoption.ItemFor(
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
			"primaryWorkerId":   selectoption.OptionalIDMeta(entity.PrimaryWorkerID),
			"secondaryWorkerId": selectoption.OptionalIDMeta(entity.SecondaryWorkerID),
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

func (r *Sources) FleetCode(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.FleetCodeService.GetByIDs(
			ctx,
			fleetcodeports.GetFleetCodesByIDsRequest{
				TenantInfo:   req.TenantInfo,
				FleetCodeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*fleetcode.FleetCode).GetID,
			fleetCodeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.FleetCodeService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		fleetCodeSelectOptionItem,
	)
}

func fleetCodeSelectOptionItem(entity *fleetcode.FleetCode) selectoption.Item {
	return selectoption.ItemFor(
		fleetCodeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func fleetCodeSelectOption(entity *fleetcode.FleetCode) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
			"color":                  entity.Color,
		},
	}
}

func equipmentTypeClassesFilter(filters map[string]any) []string {
	value, ok := filters["classes"]
	if !ok {
		value, ok = filters["class"]
	}
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []string:
		return typed
	case []any:
		classes := make([]string, 0, len(typed))
		for _, item := range typed {
			class, isString := item.(string)
			if isString && class != "" {
				classes = append(classes, class)
			}
		}
		return classes
	default:
		return nil
	}
}
