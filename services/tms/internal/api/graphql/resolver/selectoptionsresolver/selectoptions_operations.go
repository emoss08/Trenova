package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveFleetCodeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.FleetCodeService.Get(
				ctx,
				fleetcodeports.GetFleetCodeByIDRequest{
					ID:         id,
					TenantInfo: &req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, fleetCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.FleetCodeService.SelectOptions(ctx, req.SelectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		fleetCodeSelectOptionItem,
	)
}

func (r *Deps) resolveShipmentTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.ShipmentTypeService.Get(
				ctx,
				shipmenttypeports.GetShipmentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, shipmentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ShipmentTypeService.SelectOptions(
		ctx,
		&shipmenttypeports.ShipmentTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		shipmentTypeSelectOptionItem,
	)
}

func (r *Deps) resolveServiceTypeSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.ServiceTypeService.Get(
				ctx,
				servicetypeports.GetServiceTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, serviceTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ServiceTypeService.SelectOptions(
		ctx,
		&servicetypeports.ServiceTypeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		serviceTypeSelectOptionItem,
	)
}

func (r *Deps) resolveDistanceProfileSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DistanceProfileService.Get(
				ctx,
				distanceprofileports.GetDistanceProfileByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, distanceProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
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

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		distanceProfileSelectOptionItem,
	)
}

func (r *Deps) resolveCommoditySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.CommodityService.Get(
				ctx,
				commodityports.GetCommodityByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, commoditySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.CommodityService.SelectOptions(
		ctx,
		&commodityports.CommoditySelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		commoditySelectOptionItem,
	)
}

func (r *Deps) resolveHazardousMaterialSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.HazardousMaterialService.Get(
				ctx,
				hazardousmaterialports.GetHazardousMaterialByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, hazardousMaterialSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.HazardousMaterialService.SelectOptions(
		ctx,
		&hazardousmaterialports.HazardousMaterialSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
		hazardousMaterialSelectOptionItem,
	)
}

func (r *Deps) resolveOrderSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.OrderService.GetByIDs(
			ctx,
			orderports.GetOrdersByIDsRequest{
				TenantInfo: req.TenantInfo,
				OrderIDs:   req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.IDs, entities, orderID, orderSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.OrderService.SelectOptions(
		ctx,
		&orderports.OrderSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			AttachableOnly:     selectOptionBoolFilter(req.Filters, "attachableOnly"),
			CustomerID:         selectOptionCustomerFilter(req.Filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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

func (r *Deps) resolveShipmentSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.ShipmentService.GetByIDs(
			ctx,
			&shipmentports.GetShipmentsByIDsRequest{
				TenantInfo:  req.TenantInfo,
				ShipmentIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.IDs, entities, shipmentID, shipmentSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ShipmentService.SelectOptions(
		ctx,
		&shipmentports.ShipmentSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			CustomerID:         selectOptionCustomerFilter(req.Filters),
			AttachableOnly:     selectOptionBoolFilter(req.Filters, "attachableOnly"),
			ExcludeOrderID:     selectOptionIDFilter(req.Filters, "excludeOrderId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.SelectQuery.Pagination.SafeOffset(),
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
