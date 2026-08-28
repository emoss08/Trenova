package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
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
		gqlmodel.SelectOptionResourceCarrier: {
			resolve: r.resolveCarrierSelectOptions,
		},
		gqlmodel.SelectOptionResourceCustomer: {
			resolve: r.resolveCustomerSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiConnection: {
			resolve: r.resolveEDIConnectionSelectOptions,
		},
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
		gqlmodel.SelectOptionResourceFuelIndex: {
			resolve: r.resolveFuelIndexSelectOptions,
		},
		gqlmodel.SelectOptionResourceFuelSurchargeProgram: {
			resolve: r.resolveFuelSurchargeProgramSelectOptions,
		},
		gqlmodel.SelectOptionResourceFiscalYear: {
			resolve: r.resolveFiscalYearSelectOptions,
		},
		gqlmodel.SelectOptionResourceFiscalPeriod: {
			resolve: r.resolveFiscalPeriodSelectOptions,
		},
		gqlmodel.SelectOptionResourceGlAccount: {
			resolve: r.resolveGLAccountSelectOptions,
		},
		gqlmodel.SelectOptionResourceLocation: {
			resolve: r.resolveLocationSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateZone: {
			resolve: r.resolveRateZoneSelectOptions,
		},
		gqlmodel.SelectOptionResourceFleetCode: {
			resolve: r.resolveFleetCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourceShipmentType: {
			resolve: r.resolveShipmentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceServiceType: {
			resolve: r.resolveServiceTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceLocationCategory: {
			resolve: r.resolveLocationCategorySelectOptions,
		},
		gqlmodel.SelectOptionResourceDistanceProfile: {
			resolve: r.resolveDistanceProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceOrganization: {
			resolve: r.resolveOrganizationSelectOptions,
		},
		gqlmodel.SelectOptionResourceUser: {
			resolve: r.resolveUserSelectOptions,
		},
		gqlmodel.SelectOptionResourceRole: {
			resolve: r.resolveRoleSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateMatrix: {
			resolve: r.resolveRateMatrixSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateAgreement: {
			resolve: r.resolveRateAgreementSelectOptions,
		},
		gqlmodel.SelectOptionResourceAccessorialCharge: {
			resolve: r.resolveAccessorialChargeSelectOptions,
		},
		gqlmodel.SelectOptionResourceAccountType: {
			resolve: r.resolveAccountTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceCommodity: {
			resolve: r.resolveCommoditySelectOptions,
		},
		gqlmodel.SelectOptionResourceDocumentType: {
			resolve: r.resolveDocumentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceDetentionPolicy: {
			resolve: r.resolveDetentionPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceFormulaTemplate: {
			resolve: r.resolveFormulaTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceHazardousMaterial: {
			resolve: r.resolveHazardousMaterialSelectOptions,
		},
		gqlmodel.SelectOptionResourceServiceFailureReasonCode: {
			resolve: r.resolveServiceFailureReasonCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiCommunicationProfile: {
			resolve: r.resolveEDICommunicationProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiDocumentType: {
			resolve: r.resolveEDIDocumentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiMappingProfile: {
			resolve: r.resolveEDIMappingProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiPartner: {
			resolve: r.resolveEDIPartnerSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiPartnerDocumentProfile: {
			resolve: r.resolveEDIPartnerDocumentProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEdiTemplate: {
			resolve: r.resolveEDITemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceEmailProfile: {
			resolve: r.resolveEmailProfileSelectOptions,
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
			OwnerOperatorsOnly: selectOptionBoolFilter(req.filters, "ownerOperatorsOnly"),
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

func (r *Resolver) resolveLocationSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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

func (r *Resolver) resolveFleetCodeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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
	req selectOptionsRequest,
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

func (r *Resolver) resolveLocationCategorySelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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

func (r *Resolver) resolveDistanceProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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

func (r *Resolver) resolveOrganizationSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.organizationService.GetByIDs(
			ctx,
			portservices.GetOrganizationsByIDsRequest{
				TenantInfo:      req.tenantInfo,
				OrganizationIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(req.ids, entities, organizationID, organizationSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.organizationService.SelectOptions(
		ctx,
		&repositories.SelectOrganizationOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			Scope:              selectOptionStringFilter(req.filters, "scope"),
			ExcludeCurrent:     selectOptionBoolFilter(req.filters, "excludeCurrent"),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		organizationSelectOptionItem,
	)
}

func (r *Resolver) resolveUserSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.userService.GetByID(
				ctx,
				repositories.GetUserByIDRequest{
					TenantInfo:   req.tenantInfo,
					LookupUserID: id,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, userSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.userService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		userSelectOptionItem,
	)
}

func (r *Resolver) resolveRoleSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.roleService.GetRoleByID(
				ctx,
				repositories.GetRoleByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, roleSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.roleService.SelectRoleOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		roleSelectOptionItem,
	)
}

func (r *Resolver) resolveRateMatrixSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.rateMatrixService.GetByID(
				ctx,
				&repositories.GetRateMatrixByIDRequest{
					RateMatrixID: id,
					TenantInfo:   req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateMatrixSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.rateMatrixService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		rateMatrixSelectOptionItem,
	)
}

func (r *Resolver) resolveRateAgreementSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.rateAgreementService.GetByID(
				ctx,
				&repositories.GetRateAgreementByIDRequest{
					RateAgreementID: id,
					TenantInfo:      req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, rateAgreementSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.rateAgreementService.SelectOptions(ctx, req.selectQuery)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		rateAgreementSelectOptionItem,
	)
}

func (r *Resolver) resolveAccessorialChargeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.accessorialChargeService.Get(
				ctx,
				repositories.GetAccessorialChargeByIDRequest{
					ID:         id,
					TenantInfo: &req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accessorialChargeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.accessorialChargeService.SelectOptions(
		ctx,
		req.selectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		accessorialChargeSelectOptionItem,
	)
}

func (r *Resolver) resolveAccountTypeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.accountTypeService.Get(
				ctx,
				repositories.GetAccountTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, accountTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.accountTypeService.SelectOptions(
		ctx,
		&repositories.AccountTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		accountTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveCommoditySelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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

func (r *Resolver) resolveDocumentTypeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.documentTypeService.Get(
				ctx,
				repositories.GetDocumentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, documentTypeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.documentTypeService.SelectOptions(
		ctx,
		req.selectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		documentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveDetentionPolicySelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.detentionPolicyService.GetByID(
				ctx,
				&repositories.GetDetentionPolicyByIDRequest{
					DetentionPolicyID: id,
					TenantInfo:        req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, detentionPolicySelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.detentionPolicyService.SelectOptions(
		ctx,
		&repositories.DetentionPolicySelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		detentionPolicySelectOptionItem,
	)
}

func (r *Resolver) resolveFormulaTemplateSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.formulaTemplateService.GetByID(
				ctx,
				repositories.GetFormulaTemplateByIDRequest{
					TemplateID: id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, formulaTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.formulaTemplateService.SelectOptions(
		ctx,
		&repositories.FormulaTemplateSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		formulaTemplateSelectOptionItem,
	)
}

func (r *Resolver) resolveHazardousMaterialSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
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

func (r *Resolver) resolveServiceFailureReasonCodeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.serviceFailureReasonCodeSvc.Get(
				ctx,
				repositories.GetServiceFailureReasonCodeByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, serviceFailureReasonCodeSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.serviceFailureReasonCodeSvc.SelectOptions(
		ctx,
		&repositories.ServiceFailureReasonCodeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		serviceFailureReasonCodeSelectOptionItem,
	)
}

func (r *Resolver) resolveEDICommunicationProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetCommunicationProfile(
				ctx,
				repositories.GetEDICommunicationProfileByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediCommunicationProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectCommunicationProfileOptions(
		ctx,
		&repositories.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediCommunicationProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIDocumentTypeSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		result, err := r.ediService.SelectDocumentTypeOptions(
			ctx,
			&repositories.EDIDocumentTypeSelectOptionsRequest{
				SelectQueryRequest: req.selectQuery,
			},
		)
		if err != nil {
			return nil, err
		}
		items := orderedSelectOptionItems(req.ids, result.Items, func(e *edi.EDIDocumentType) pulid.ID { return e.ID }, ediDocumentTypeSelectOptionItem)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectDocumentTypeOptions(
		ctx,
		&repositories.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediDocumentTypeSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIMappingProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetMappingProfileByID(
				ctx,
				repositories.GetMappingProfileByIDRequest{
					ProfileID:  id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediMappingProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectMappingProfileOptions(
		ctx,
		&repositories.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediMappingProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIPartnerSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetPartner(
				ctx,
				repositories.GetEDIPartnerByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectPartnerOptions(
		ctx,
		&repositories.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediPartnerSelectOptionItem,
	)
}

func (r *Resolver) resolveEDIPartnerDocumentProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetPartnerDocumentProfile(
				ctx,
				repositories.GetEDIPartnerDocumentProfileByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediPartnerDocumentProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectPartnerDocumentProfileOptions(
		ctx,
		&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			PartnerID:          selectOptionIDFilter(req.filters, "partnerId"),
			TransactionSet:     edi.TransactionSet(selectOptionStringFilter(req.filters, "transactionSet")),
			Direction:          edi.DocumentDirection(selectOptionStringFilter(req.filters, "direction")),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediPartnerDocumentProfileSelectOptionItem,
	)
}

func (r *Resolver) resolveEDITemplateSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.ediService.GetTemplate(
				ctx,
				repositories.GetEDITemplateByIDRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, ediTemplateSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.SelectTemplateOptions(
		ctx,
		&repositories.EDITemplateSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
			TransactionSet:     edi.TransactionSet(selectOptionStringFilter(req.filters, "transactionSet")),
			Direction:          edi.DocumentDirection(selectOptionStringFilter(req.filters, "direction")),
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediTemplateSelectOptionItem,
	)
}

func (r *Resolver) resolveEmailProfileSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.emailService.GetProfile(
				ctx,
				repositories.GetEmailEntityRequest{
					ID:         id,
					TenantInfo: req.tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, emailProfileSelectOptionItem(entity))
		}

		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.emailService.SelectProfileOptions(
		ctx,
		&repositories.EmailProfileSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		emailProfileSelectOptionItem,
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

// selectOptionCustomerFilter extracts an optional customerId scope from select-option
// filters (used to restrict the shipment picker to an order's customer).
func selectOptionCustomerFilter(filters map[string]any) pulid.ID {
	return selectOptionIDFilter(filters, "customerId")
}

func selectOptionIDFilter(filters map[string]any, key string) pulid.ID {
	value, ok := filters[key]
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

func selectOptionBoolFilter(filters map[string]any, key string) bool {
	value, ok := filters[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}

func selectOptionStringFilter(filters map[string]any, key string) string {
	value, ok := filters[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

func (r *Resolver) resolveEDIConnectionSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		entities, err := r.ediService.GetConnectionsByIDs(
			ctx,
			repositories.GetEDIConnectionsByIDsRequest{
				TenantInfo:    req.tenantInfo,
				ConnectionIDs: req.ids,
			},
		)
		if err != nil {
			return nil, err
		}

		items := orderedSelectOptionItems(
			req.ids,
			entities,
			ediConnectionID,
			ediConnectionSelectOptionItem,
		)
		return selectOptionConnection(items, len(items), 0)
	}

	result, err := r.ediService.ConnectionSelectOptions(
		ctx,
		&repositories.EDIConnectionSelectOptionsRequest{
			SelectQueryRequest: req.selectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectOptionListConnection(
		result,
		req.selectQuery.Pagination.SafeOffset(),
		ediConnectionSelectOptionItem,
	)
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
		Description: stringPtr(entity.Code),
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
		Description: stringPtr(entity.Code),
		Meta: map[string]any{
			"code":   entity.Code,
			"status": string(entity.Status),
		},
	}
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
		Description: stringPtr(entity.Description),
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
		Description: stringPtr(entity.Description),
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
		Description: stringPtr(entity.Description),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
		},
	}
}

func locationCategorySelectOptionItem(entity *locationcategory.LocationCategory) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		locationCategorySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func locationCategorySelectOption(entity *locationcategory.LocationCategory) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
		Meta: map[string]any{
			"color": entity.Color,
		},
	}
}

func distanceProfileSelectOptionItem(entity *distanceprofile.DistanceProfile) selectOptionConnectionItem {
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
		Description: stringPtr(entity.RoutingType + " \u00b7 " + entity.DistanceUnits),
		Meta: map[string]any{
			"routingType":   entity.RoutingType,
			"distanceUnits": entity.DistanceUnits,
			"isDefault":     entity.IsDefault,
		},
	}
}

func organizationSelectOptionItem(entity *tenant.Organization) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		organizationSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func organizationSelectOption(entity *tenant.Organization) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.ScacCode),
		Meta: map[string]any{
			"scacCode": entity.ScacCode,
			"city":     entity.City,
		},
	}
}

func organizationID(entity *tenant.Organization) pulid.ID {
	return entity.ID
}

func userSelectOptionItem(entity *tenant.User) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		userSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func userSelectOption(entity *tenant.User) *gqlmodel.SelectOption {
	label := entity.Name
	if label == "" {
		label = entity.EmailAddress
	}
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringPtr(entity.EmailAddress),
		Meta: map[string]any{
			"name":         entity.Name,
			"email":        entity.EmailAddress,
			"emailAddress": entity.EmailAddress,
		},
	}
}

func userID(entity *tenant.User) pulid.ID {
	return entity.ID
}

func roleSelectOptionItem(entity *permission.Role) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		roleSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func roleSelectOption(entity *permission.Role) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
	}
}

func roleID(entity *permission.Role) pulid.ID {
	return entity.ID
}

func rateMatrixSelectOptionItem(entity *ratematrix.RateMatrix) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		rateMatrixSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func rateMatrixSelectOption(entity *ratematrix.RateMatrix) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func rateAgreementSelectOptionItem(entity *rateagreement.RateAgreement) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		rateAgreementSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func rateAgreementSelectOption(entity *rateagreement.RateAgreement) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func accessorialChargeSelectOptionItem(entity *accessorialcharge.AccessorialCharge) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		accessorialChargeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func accessorialChargeSelectOption(entity *accessorialcharge.AccessorialCharge) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringPtr(entity.Description),
		Meta: map[string]any{
			"code":   entity.Code,
			"method": string(entity.Method),
			"amount": entity.Amount.String(),
		},
	}
}

func accountTypeSelectOptionItem(entity *accounttype.AccountType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		accountTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func accountTypeSelectOption(entity *accounttype.AccountType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringPtr(entity.Name),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
			"name":  entity.Name,
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
		Description: stringPtr(entity.Description),
	}
}

func documentTypeSelectOptionItem(entity *documenttype.DocumentType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		documentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func documentTypeSelectOption(entity *documenttype.DocumentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringPtr(entity.Name),
		Meta: map[string]any{
			"code":  entity.Code,
			"color": entity.Color,
			"name":  entity.Name,
		},
	}
}

func detentionPolicySelectOptionItem(entity *detention.DetentionPolicy) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		detentionPolicySelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func detentionPolicySelectOption(entity *detention.DetentionPolicy) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Code),
		Meta: map[string]any{
			"code": entity.Code,
		},
	}
}

func formulaTemplateSelectOptionItem(entity *formulatemplate.FormulaTemplate) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		formulaTemplateSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func formulaTemplateSelectOption(entity *formulatemplate.FormulaTemplate) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
	}
}

func hazardousMaterialSelectOptionItem(entity *hazardousmaterial.HazardousMaterial) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		hazardousMaterialSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func hazardousMaterialSelectOption(entity *hazardousmaterial.HazardousMaterial) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
		Meta: map[string]any{
			"class": entity.Class,
		},
	}
}

func serviceFailureReasonCodeSelectOptionItem(entity *servicefailure.ReasonCode) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		serviceFailureReasonCodeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func serviceFailureReasonCodeSelectOption(entity *servicefailure.ReasonCode) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringPtr(entity.Label),
		Meta: map[string]any{
			"code":  entity.Code,
			"label": entity.Label,
		},
	}
}

func ediCommunicationProfileSelectOptionItem(entity *edi.EDICommunicationProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediCommunicationProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediCommunicationProfileSelectOption(entity *edi.EDICommunicationProfile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(string(entity.Method)),
		Meta: map[string]any{
			"method": string(entity.Method),
			"status": string(entity.Status),
		},
	}
}

func ediDocumentTypeSelectOptionItem(entity *edi.EDIDocumentType) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediDocumentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediDocumentTypeSelectOption(entity *edi.EDIDocumentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code + " - " + entity.Name,
		Description: stringPtr(string(entity.TransactionSet) + " / " + string(entity.Direction)),
		Meta: map[string]any{
			"code":           entity.Code,
			"transactionSet": string(entity.TransactionSet),
			"direction":      string(entity.Direction),
			"defaultVersion": entity.DefaultVersion,
		},
	}
}

func ediMappingProfileSelectOptionItem(entity *edi.EDIMappingProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediMappingProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediMappingProfileSelectOption(entity *edi.EDIMappingProfile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
	}
}

func ediPartnerSelectOptionItem(entity *edi.EDIPartner) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediPartnerSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediPartnerSelectOption(entity *edi.EDIPartner) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code + " - " + entity.Name,
		Description: stringPtr(string(entity.Kind)),
		Meta: map[string]any{
			"code": entity.Code,
			"kind": string(entity.Kind),
		},
	}
}

func ediPartnerDocumentProfileSelectOptionItem(entity *edi.EDIPartnerDocumentProfile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediPartnerDocumentProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediPartnerDocumentProfileSelectOption(entity *edi.EDIPartnerDocumentProfile) *gqlmodel.SelectOption {
	label := entity.Name
	if entity.Partner != nil {
		label = entity.Partner.Code + " - " + entity.Name
	}
	description := string(entity.TransactionSet) + " \u00b7 " + string(entity.Direction)
	if entity.Standard != "" {
		description = string(entity.Standard) + " \u00b7 " + description
	}
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringPtr(description),
		Meta: map[string]any{
			"transactionSet": string(entity.TransactionSet),
			"direction":      string(entity.Direction),
			"standard":       string(entity.Standard),
			"status":         string(entity.Status),
		},
	}
}

func ediTemplateSelectOptionItem(entity *edi.EDITemplate) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediTemplateSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediTemplateSelectOption(entity *edi.EDITemplate) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.Description),
		Meta: map[string]any{
			"status": string(entity.Status),
		},
	}
}

func emailProfileSelectOptionItem(entity *email.Profile) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		emailProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func emailProfileSelectOption(entity *email.Profile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringPtr(entity.SenderEmail),
		Meta: map[string]any{
			"senderEmail": entity.SenderEmail,
			"provider":    string(entity.Provider),
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

func ediConnectionSelectOptionItem(entity *edi.EDIConnection) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediConnectionSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediConnectionSelectOption(entity *edi.EDIConnection) *gqlmodel.SelectOption {
	source := ""
	if entity.SourceOrganization != nil {
		source = entity.SourceOrganization.Name
	}
	if source == "" {
		source = entity.SourceOrganizationID.String()
	}
	target := ""
	if entity.TargetOrganization != nil {
		target = entity.TargetOrganization.Name
	}
	if target == "" {
		target = entity.TargetOrganizationID.String()
	}
	label := source + " → " + target

	description := string(entity.Method) + " \u00b7 " + string(entity.Status)

	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       label,
		Description: stringPtr(description),
		Meta: map[string]any{
			"method":                 string(entity.Method),
			"status":                 string(entity.Status),
			"sourceOrganizationId":   entity.SourceOrganizationID.String(),
			"targetOrganizationId":   entity.TargetOrganizationID.String(),
			"sourceOrganizationName": source,
			"targetOrganizationName": target,
			"sourcePartnerId":        optionalIDString(entity.SourcePartnerID),
			"targetPartnerId":        optionalIDString(entity.TargetPartnerID),
		},
	}
}

func ediConnectionID(entity *edi.EDIConnection) pulid.ID {
	return entity.ID
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

func locationID(entity *location.Location) pulid.ID {
	return entity.ID
}

func shipmentTypeID(entity *shipmenttype.ShipmentType) pulid.ID {
	return entity.ID
}

func serviceTypeID(entity *servicetype.ServiceType) pulid.ID {
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
