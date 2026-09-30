package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Resolver) resolveFleetCodeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.fleetCodeService.Get(
				ctx,
				repositories.GetFleetCodeByIDRequest{
					ID:         id,
					TenantInfo: &req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, fleetCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.fleetCodeService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		fleetCodeSelectOptionItem,
	)
}

func (r *Resolver) resolveShipmentTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.shipmentTypeService.Get(
				ctx,
				repositories.GetShipmentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, shipmentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.shipmentTypeService.SelectOptions(
		ctx,
		&repositories.ShipmentTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		shipmentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveServiceTypeSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.serviceTypeService.Get(
				ctx,
				repositories.GetServiceTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, serviceTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.serviceTypeService.SelectOptions(
		ctx,
		&repositories.ServiceTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		serviceTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveDistanceProfileSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.distanceProfileService.Get(
				ctx,
				repositories.GetDistanceProfileByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, distanceProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.distanceProfileService.SelectOptions(
		ctx,
		&repositories.DistanceProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		distanceProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveCommoditySelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.commodityService.Get(
				ctx,
				repositories.GetCommodityByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, commoditySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.commodityService.SelectOptions(
		ctx,
		&repositories.CommoditySelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		commoditySelectOptionItem,
	)
}

func (r *Resolver) resolveHazardousMaterialSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.hazardousMaterialService.Get(
				ctx,
				repositories.GetHazardousMaterialByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, hazardousMaterialSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.hazardousMaterialService.SelectOptions(
		ctx,
		&repositories.HazardousMaterialSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		hazardousMaterialSelectOptionItem,
	)
}

func (r *Resolver) resolveOrderSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.orderService.GetByIDs(
			ctx,
			repositories.GetOrdersByIDsRequest{
				TenantInfo: req.tenantInfo,
				OrderIDs:   req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, orderID, orderSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.orderService.SelectOptions(
		ctx,
		&repositories.OrderSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			AttachableOnly:     selectOptionBoolFilter(req.filters, "attachableOnly"),
			CustomerID:         selectOptionCustomerFilter(req.filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		orderSelectOptionItem,
	)
}

func orderSelectOption(entity *order.Order) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.OrderNumber,
		Description: stringutils.Ptr(entity.PONumber),
		Meta: map[string]any{
			"status":      string(entity.Status),
			"orderNumber": entity.OrderNumber,
		},
	}
}

func orderSelectOptionItem(entity *order.Order) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		orderSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func orderID(entity *order.Order) pulid.ID {
	return entity.ID
}

func (r *Resolver) resolveShipmentSelectOptions(
	ctx context.Context,
	req *selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.shipmentService.GetByIDs(
			ctx,
			&repositories.GetShipmentsByIDsRequest{
				TenantInfo:  req.tenantInfo,
				ShipmentIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, shipmentID, shipmentSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.shipmentService.SelectOptions(
		ctx,
		&repositories.ShipmentSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			CustomerID:         selectOptionCustomerFilter(req.filters),
			AttachableOnly:     selectOptionBoolFilter(req.filters, "attachableOnly"),
			ExcludeOrderID:     selectOptionIDFilter(req.filters, "excludeOrderId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		shipmentSelectOptionItem,
	)
}

func fleetCodeSelectOptionItem(entity *fleetcode.FleetCode) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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
			"code":  entity.Code,
			"color": entity.Color,
		},
	}
}

func shipmentTypeSelectOptionItem(entity *shipmenttype.ShipmentType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		shipmentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func shipmentTypeSelectOption(entity *shipmenttype.ShipmentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
		},
	}
}

func serviceTypeSelectOptionItem(entity *servicetype.ServiceType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		serviceTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func serviceTypeSelectOption(entity *servicetype.ServiceType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
		},
	}
}

func distanceProfileSelectOptionItem(
	entity *distanceprofile.DistanceProfile,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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

func commoditySelectOptionItem(entity *commodity.Commodity) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		commoditySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func commoditySelectOption(entity *commodity.Commodity) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
	}
}

func hazardousMaterialSelectOptionItem(
	entity *hazardousmaterial.HazardousMaterial,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		hazardousMaterialSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func hazardousMaterialSelectOption(
	entity *hazardousmaterial.HazardousMaterial,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"class": entity.Class,
		},
	}
}

func shipmentSelectOptionItem(entity *shipment.Shipment) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		shipmentSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func shipmentSelectOption(entity *shipment.Shipment) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.ProNumber,
		Description: stringutils.Ptr(entity.BOL),
		Meta: map[string]any{
			"status":    string(entity.Status),
			"proNumber": entity.ProNumber,
			"bol":       entity.BOL,
		},
	}
}

func shipmentID(entity *shipment.Shipment) pulid.ID {
	return entity.ID
}

func shipmentTypeID(entity *shipmenttype.ShipmentType) pulid.ID {
	return entity.ID
}

func serviceTypeID(entity *servicetype.ServiceType) pulid.ID {
	return entity.ID
}
