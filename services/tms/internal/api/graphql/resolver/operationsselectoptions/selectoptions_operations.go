package operationsselectoptions

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/servicetypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Sources struct {
	ShipmentTypeService         *shipmenttypeservice.Service
	ServiceTypeService          *servicetypeservice.Service
	CommodityService            *commodityservice.Service
	HazardousMaterialService    *hazardousmaterialservice.Service
	OrderService                *orderservice.Service
	ShipmentReader              shipmentserviceports.ShipmentReader
	ServiceFailureReasonCodeSvc servicefailureports.ServiceFailureReasonCodeService
}

func (r *Sources) ShipmentType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.ShipmentTypeService.GetByIDs(
			ctx,
			shipmenttypeports.GetShipmentTypesByIDsRequest{
				TenantInfo:      req.TenantInfo,
				ShipmentTypeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*shipmenttype.ShipmentType).GetID,
			shipmentTypeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
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

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		shipmentTypeSelectOptionItem,
	)
}

func (r *Sources) ServiceType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.ServiceTypeService.GetByIDs(
			ctx,
			servicetypeports.GetServiceTypesByIDsRequest{
				TenantInfo:     req.TenantInfo,
				ServiceTypeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*servicetype.ServiceType).GetID,
			serviceTypeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
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

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		serviceTypeSelectOptionItem,
	)
}

func (r *Sources) Commodity(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.CommodityService.GetByIDs(
			ctx,
			commodityports.GetCommoditiesByIDsRequest{
				TenantInfo:   req.TenantInfo,
				CommodityIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*commodity.Commodity).GetID,
			commoditySelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
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

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		commoditySelectOptionItem,
	)
}

func (r *Sources) HazardousMaterial(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.HazardousMaterialService.GetByIDs(
			ctx,
			hazardousmaterialports.GetHazardousMaterialsByIDsRequest{
				TenantInfo:           req.TenantInfo,
				HazardousMaterialIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*hazardousmaterial.HazardousMaterial).GetID,
			hazardousMaterialSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
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

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		hazardousMaterialSelectOptionItem,
	)
}

func (r *Sources) Order(
	ctx context.Context,
	req *selectoption.Request,
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

		items := selectoption.OrderedItems(req.IDs, entities, orderID, orderSelectOptionItem)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.OrderService.SelectOptions(
		ctx,
		&orderports.OrderSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			AttachableOnly:     selectoption.BoolFilter(req.Filters, "attachableOnly"),
			CustomerID:         selectoption.CustomerFilter(req.Filters),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		orderSelectOptionItem,
	)
}

func orderSelectOption(entity *order.Order) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.OrderNumber,
		Description: stringutils.Ptr(entity.PONumber),
		Meta: map[string]any{
			selectoption.MetaKeyStatus: string(entity.Status),
			"orderNumber":              entity.OrderNumber,
		},
	}
}

func orderSelectOptionItem(entity *order.Order) selectoption.Item {
	return selectoption.ItemFor(
		orderSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func orderID(entity *order.Order) pulid.ID {
	return entity.ID
}

func (r *Sources) Shipment(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.ShipmentReader.GetByIDs(
			ctx,
			&shipmentports.GetShipmentsByIDsRequest{
				TenantInfo:  req.TenantInfo,
				ShipmentIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(req.IDs, entities, shipmentID, shipmentSelectOptionItem)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.ShipmentReader.SelectOptions(
		ctx,
		&shipmentports.ShipmentSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
			CustomerID:         selectoption.CustomerFilter(req.Filters),
			AttachableOnly:     selectoption.BoolFilter(req.Filters, "attachableOnly"),
			ExcludeOrderID:     selectoption.IDFilter(req.Filters, "excludeOrderId"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		shipmentSelectOptionItem,
	)
}

func shipmentTypeSelectOptionItem(entity *shipmenttype.ShipmentType) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
			"color":                  entity.Color,
		},
	}
}

func serviceTypeSelectOptionItem(entity *servicetype.ServiceType) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyCode: entity.Code,
			"color":                  entity.Color,
		},
	}
}

func commoditySelectOptionItem(entity *commodity.Commodity) selectoption.Item {
	return selectoption.ItemFor(
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
		Meta: map[string]any{
			"hazardousMaterialId": entity.HazardousMaterialID.String(),
			"stackable":           entity.Stackable,
			"fragile":             entity.Fragile,
		},
	}
}

func hazardousMaterialSelectOptionItem(
	entity *hazardousmaterial.HazardousMaterial,
) selectoption.Item {
	return selectoption.ItemFor(
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

func shipmentSelectOptionItem(entity *shipment.Shipment) selectoption.Item {
	return selectoption.ItemFor(
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
			selectoption.MetaKeyStatus: string(entity.Status),
			"proNumber":                entity.ProNumber,
			"bol":                      entity.BOL,
		},
	}
}

func shipmentID(entity *shipment.Shipment) pulid.ID {
	return entity.ID
}

func (r *Sources) ServiceFailureReasonCode(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		entities, err := r.ServiceFailureReasonCodeSvc.GetByIDs(
			ctx,
			servicefailureports.GetServiceFailureReasonCodesByIDsRequest{
				TenantInfo:    req.TenantInfo,
				ReasonCodeIDs: req.IDs,
			},
		)
		if err != nil {
			return nil, err
		}

		items := selectoption.OrderedItems(
			req.IDs,
			entities,
			(*servicefailure.ReasonCode).GetID,
			serviceFailureReasonCodeSelectOptionItem,
		)
		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.ServiceFailureReasonCodeSvc.SelectOptions(
		ctx,
		&servicefailureports.ServiceFailureReasonCodeSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		serviceFailureReasonCodeSelectOptionItem,
	)
}
