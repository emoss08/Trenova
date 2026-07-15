package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type selectOptionRegistryEntry struct {
	resolve func(context.Context, selectOptionsRequest) (*gqlmodel.SelectOptionConnection, error)
}

type selectOptionsRequest struct {
	tenantInfo  pagination.TenantInfo
	selectQuery *pagination.SelectQueryRequest
	ids         []pulid.ID
	filters     map[string]any
}

type selectOptionConnectionItem struct {
	option *gqlmodel.SelectOption
	cursor pagination.Cursor
}

func (r *Resolver) requireAuthContext(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

func (r *queryResolver) resolveSelectOptions(
	ctx context.Context,
	input gqlmodel.SelectOptionsInput,
	registry map[gqlmodel.SelectOptionResource]selectOptionRegistryEntry,
) (*gqlmodel.SelectOptionConnection, error) {
	entry, ok := registry[input.Resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"Select option resource is not supported",
		)
	}

	authCtx, err := r.requireAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	req, err := selectOptionsRequestFromInput(input, authCtx)
	if err != nil {
		return nil, err
	}

	return entry.resolve(ctx, req)
}

func (r *Resolver) selectOptionRegistry() map[gqlmodel.SelectOptionResource]selectOptionRegistryEntry {
	return map[gqlmodel.SelectOptionResource]selectOptionRegistryEntry{
		gqlmodel.SelectOptionResourceEquipmentType: {
			resolve: r.resolveEquipmentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceEquipmentManufacturer: {
			resolve: r.resolveEquipmentManufacturerSelectOptions,
		},
		gqlmodel.SelectOptionResourceTrailer: {
			resolve: r.resolveTrailerSelectOptions,
		},
		gqlmodel.SelectOptionResourceTractor: {
			resolve: r.resolveTractorSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorker: {
			resolve: r.resolveWorkerSelectOptions,
		},
		gqlmodel.SelectOptionResourceUsState: {
			resolve: r.resolveUSStateSelectOptions,
		},
		gqlmodel.SelectOptionResourceShipment: {
			resolve: r.resolveShipmentSelectOptions,
		},
		gqlmodel.SelectOptionResourceOrder: {
			resolve: r.resolveOrderSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiTransfer: {
			resolve: r.resolveEDITransferSelectOptions,
		},
	}
}

func selectOptionsRequestFromInput(
	input gqlmodel.SelectOptionsInput,
	authCtx *authctx.AuthContext,
) (selectOptionsRequest, error) {
	ids, err := parseIDs(input.Ids)
	if err != nil {
		return selectOptionsRequest{}, err
	}

	first := pagination.DefaultLimit
	if input.First != nil {
		first = pagination.ClampLimit(*input.First)
	}

	offset := pagination.DefaultOffset
	if input.Offset != nil {
		offset = pagination.ClampOffset(*input.Offset)
	}

	tenant := tenantInfo(authCtx)
	return selectOptionsRequest{
		tenantInfo: tenant,
		ids:        ids,
		filters:    selectOptionFilters(input.Filters),
		selectQuery: &pagination.SelectQueryRequest{
			TenantInfo: tenant,
			Pagination: pagination.Info{
				Limit:  first,
				Offset: offset,
			},
			Query: stringValue(input.Query),
		},
	}, nil
}

func selectOptionFilters(filters map[string]any) map[string]any {
	if len(filters) == 0 {
		return map[string]any{}
	}

	return filters
}

func (r *Resolver) resolveEquipmentTypeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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

func (r *Resolver) resolveWorkerSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.workerService.Get(
				ctx,
				repositories.GetWorkerByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, workerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.workerService.SelectOptions(
		ctx,
		&repositories.WorkerSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		workerSelectOptionItem,
	)
}

func (r *Resolver) resolveUSStateSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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

func (r *Resolver) resolveOrderSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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
		Description: stringPtr(entity.PONumber),
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
	req selectOptionsRequest,
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

// selectOptionCustomerFilter extracts an optional customerId scope from select-option
// filters (used to restrict the shipment picker to an order's customer).
func selectOptionCustomerFilter(filters map[string]any) pulid.ID {
	value, ok := filters["customerId"]
	if !ok {
		return pulid.Nil
	}
	str, ok := value.(string)
	if !ok || str == "" {
		return pulid.Nil
	}
	id, err := pulid.Parse(str)
	if err != nil {
		return pulid.Nil
	}
	return id
}

func (r *Resolver) resolveEDITransferSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.ediService.GetTransfersByIDs(
			ctx,
			repositories.GetEDITransfersByIDsRequest{
				TenantInfo:  req.tenantInfo,
				TransferIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			ediTransferID,
			ediTransferSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.TransferSelectOptions(
		ctx,
		&repositories.EDITransferSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediTransferSelectOptionItem,
	)
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
			class, ok := item.(string)
			if ok && class != "" {
				classes = append(classes, class)
			}
		}
		return classes
	default:
		return nil
	}
}

func selectOptionListConnection[T any](
	result *pagination.ListResult[T],
	offset int,
	mapper func(T) selectOptionConnectionItem,
) (*gqlmodel.SelectOptionConnection, error) {
	items := make([]selectOptionConnectionItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapper(item))
	}

	return selectOptionConnection(items, result.Total, offset)
}

func selectOptionConnection(
	items []selectOptionConnectionItem,
	total int,
	offset int,
) (*gqlmodel.SelectOptionConnection, error) {
	hasNextPage := offset+len(items) < total

	edges := make([]*gqlmodel.SelectOptionEdge, len(items))
	for i, item := range items {
		cursor, err := pagination.EncodeCursor(item.cursor)
		if err != nil {
			return nil, err
		}
		edges[i] = &gqlmodel.SelectOptionEdge{
			Node:   item.option,
			Cursor: cursor,
		}
	}

	return &gqlmodel.SelectOptionConnection{
		Edges: edges,
		PageInfo: pageInfo(
			hasNextPage,
			lastEdgeCursor(edges, func(edge *gqlmodel.SelectOptionEdge) string {
				return edge.Cursor
			}),
		),
		TotalCount: new(total),
	}, nil
}

func orderedSelectOptionItems[T any](
	ids []pulid.ID,
	entities []T,
	id func(T) pulid.ID,
	mapper func(T) selectOptionConnectionItem,
) []selectOptionConnectionItem {
	byID := make(map[pulid.ID]T, len(entities))
	for _, entity := range entities {
		byID[id(entity)] = entity
	}

	items := make([]selectOptionConnectionItem, 0, len(entities))
	for _, requestedID := range ids {
		entity, ok := byID[requestedID]
		if ok {
			items = append(items, mapper(entity))
		}
	}

	return items
}

func selectOptionConnectionItemFor(
	option *gqlmodel.SelectOption,
	createdAt int64,
	id pulid.ID,
) selectOptionConnectionItem {
	return selectOptionConnectionItem{
		option: option,
		cursor: pagination.Cursor{
			CreatedAt: createdAt,
			ID:        id,
		},
	}
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
		Description: stringPtr(entity.Description),
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
		Description: stringPtr(entity.Description),
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
			"primaryWorkerId":   optionalIDString(entity.PrimaryWorkerID),
			"secondaryWorkerId": optionalIDString(entity.SecondaryWorkerID),
		},
	}
}

func workerSelectOptionItem(entity *worker.Worker) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		workerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func workerSelectOption(entity *worker.Worker) *gqlmodel.SelectOption {
	label := entity.WholeName
	if label == "" {
		label = entity.FullName()
	}

	meta := map[string]any{
		"firstName": entity.FirstName,
		"lastName":  entity.LastName,
		"wholeName": label,
	}
	if entity.FleetCode != nil {
		meta["fleetCode"] = entity.FleetCode.Code
	}

	return &gqlmodel.SelectOption{
		ID:    entity.ID.String(),
		Label: label,
		Meta:  meta,
	}
}

func usStateSelectOptionItem(entity *usstate.UsState) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
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
		Description: stringPtr(entity.BOL),
		Meta: map[string]any{
			"status":    string(entity.Status),
			"proNumber": entity.ProNumber,
			"bol":       entity.BOL,
		},
	}
}

func ediTransferSelectOptionItem(entity *edi.EDITransfer) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediTransferSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediTransferSelectOption(entity *edi.EDITransfer) *gqlmodel.SelectOption {
	label := entity.TenderPayload.BOL
	if label == "" {
		label = "Load tender " + entity.ID.String()
	}

	meta := map[string]any{
		"status":        string(entity.Status),
		"bol":           entity.TenderPayload.BOL,
		"customerLabel": entity.TenderPayload.CustomerLabel,
	}
	if entity.SourcePartner != nil {
		meta["sourcePartner"] = entity.SourcePartner.Name
	}
	if entity.TargetPartner != nil {
		meta["targetPartner"] = entity.TargetPartner.Name
	}

	description := entity.TenderPayload.CustomerLabel
	if description == "" {
		description = entity.TenderPayload.ServiceTypeLabel
	}

	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringPtr(description),
		Meta:        meta,
	}
}

func shipmentID(entity *shipment.Shipment) pulid.ID {
	return entity.ID
}

func ediTransferID(entity *edi.EDITransfer) pulid.ID {
	return entity.ID
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

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalIDString(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}

	return id.String()
}
