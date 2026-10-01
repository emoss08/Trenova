package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveUSStateSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.usStateService.Get(
				ctx,
				repositories.GetUsStateByIDRequest{StateID: id},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, usStateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.usStateService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		usStateSelectOptionItem,
	)
}

func (r *Resolver) resolveLocationSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.LocationService.GetByIDs(
			ctx,
			repositories.GetLocationsByIDsRequest{
				TenantInfo:  req.TenantInfo,
				LocationIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.IDs, entities, locationID, locationSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.LocationService.SelectOptions(
		ctx,
		&repositories.LocationSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		locationSelectOptionItem,
	)
}

func (r *Resolver) resolveRateZoneSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.RateZoneService.GetByID(
				ctx,
				&repositories.GetRateZoneByIDRequest{
					RateZoneID: id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateZoneSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.RateZoneService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		rateZoneSelectOptionItem,
	)
}

func (r *Resolver) resolveLocationCategorySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.LocationCategoryService.Get(
				ctx,
				repositories.GetLocationCategoryByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, locationCategorySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.LocationCategoryService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		locationCategorySelectOptionItem,
	)
}

func locationSelectOptionItem(entity *location.Location) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		locationSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func locationSelectOption(entity *location.Location) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func rateZoneSelectOptionItem(entity *ratezone.RateZone) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		rateZoneSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func rateZoneSelectOption(entity *ratezone.RateZone) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
		},
	}
}

func locationCategorySelectOptionItem(
	entity *locationcategory.LocationCategory,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		locationCategorySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func locationCategorySelectOption(
	entity *locationcategory.LocationCategory,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"color": entity.Color,
		},
	}
}

func locationID(entity *location.Location) pulid.ID {
	return entity.ID
}
