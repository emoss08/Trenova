package placesselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usstateservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/app/distanceprofileservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationcategoryservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratezoneservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	UsStateService          *usstateservice.Service
	LocationService         *locationservice.Service
	RateZoneService         *ratezoneservice.Service
	LocationCategoryService *locationcategoryservice.Service
	DistanceProfileService  *distanceprofileservice.Service
}

func (r *Sources) USState(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.UsStateService.GetByIDs(ctx, req.IDs)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*usstate.UsState).GetID,
			usStateSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.UsStateService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		usStateSelectOptionItem,
	)
}

func (r *Sources) Location(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.LocationService.GetByIDs(
			ctx,
			locationports.GetLocationsByIDsRequest{
				TenantInfo:  req.TenantInfo,
				LocationIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(req.IDs, entities, locationID, locationSelectOptionItem)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.LocationService.SelectOptions(
		ctx,
		&locationports.LocationSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		locationSelectOptionItem,
	)
}

func (r *Sources) RateZone(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.RateZoneService.GetByIDs(
			ctx,
			&ratezoneports.GetRateZonesByIDsRequest{
				TenantInfo:  req.TenantInfo,
				RateZoneIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*ratezone.RateZone).GetID,
			rateZoneSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.RateZoneService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		rateZoneSelectOptionItem,
	)
}

func (r *Sources) LocationCategory(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.LocationCategoryService.GetByIDs(
			ctx,
			locationcategoryports.GetLocationCategoriesByIDsRequest{
				TenantInfo:          req.TenantInfo,
				LocationCategoryIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*locationcategory.LocationCategory).GetID,
			locationCategorySelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.LocationCategoryService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		locationCategorySelectOptionItem,
	)
}

func locationSelectOptionItem(entity *location.Location) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
		},
	}
}

func rateZoneSelectOptionItem(entity *ratezone.RateZone) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode:   entity.Code,
			selectoption.MetaKeyStatus: string(entity.Status),
		},
	}
}

func locationCategorySelectOptionItem(
	entity *locationcategory.LocationCategory,
) selectoption.Item {
	return selectoption.ItemFor(
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

func (r *Sources) DistanceProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.DistanceProfileService.GetByIDs(
			ctx,
			distanceprofileports.GetDistanceProfilesByIDsRequest{
				TenantInfo:         req.TenantInfo,
				DistanceProfileIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*distanceprofile.DistanceProfile).GetID,
			distanceProfileSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DistanceProfileService.SelectOptions(
		ctx,
		&distanceprofileports.DistanceProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		distanceProfileSelectOptionItem,
	)
}

func distanceProfileSelectOptionItem(
	entity *distanceprofile.DistanceProfile,
) selectoption.Item {
	return selectoption.ItemFor(
		distanceProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func distanceProfileSelectOption(entity *distanceprofile.DistanceProfile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.RoutingType + " \u00b7 " + entity.DistanceUnits),
		Meta: map[string]any{
			"routingType":   entity.RoutingType,
			"distanceUnits": entity.DistanceUnits,
			"isDefault":     entity.IsDefault,
		},
	}
}

func usStateSelectOptionItem(entity *usstate.UsState) selectoption.Item {
	return selectoption.ItemFor(
		usStateSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func usStateSelectOption(entity *usstate.UsState) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: entity.Name,
		Meta: map[string]any{
			"abbreviation": entity.Abbreviation,
			"countryIso3":  entity.CountryIso3,
		},
	}
}
