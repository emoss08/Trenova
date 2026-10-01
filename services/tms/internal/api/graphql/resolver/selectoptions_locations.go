package resolver

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
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
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

	result, err := r.usStateService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		usStateSelectOptionItem,
	)
}

func (r *Resolver) resolveLocationSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.locationService.GetByIDs(
			ctx,
			repositories.GetLocationsByIDsRequest{
				TenantInfo:  req.tenantInfo,
				LocationIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, locationID, locationSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.locationService.SelectOptions(
		ctx,
		&repositories.LocationSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		locationSelectOptionItem,
	)
}

func (r *Resolver) resolveRateZoneSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.rateZoneService.GetByID(
				ctx,
				&repositories.GetRateZoneByIDRequest{
					RateZoneID: id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateZoneSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.rateZoneService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		rateZoneSelectOptionItem,
	)
}

func (r *Resolver) resolveLocationCategorySelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.locationCategoryService.Get(
				ctx,
				repositories.GetLocationCategoryByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, locationCategorySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.locationCategoryService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
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
