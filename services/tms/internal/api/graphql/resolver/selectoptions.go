package resolver

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type selectOptionRegistryEntry struct {
	resolve  func(context.Context, *selectOptionsRequest) (*gqlmodel.SelectOptionConnection, error)
	parseIDs func([]string) ([]pulid.ID, error)
}

func (e selectOptionRegistryEntry) ids(values []string) ([]pulid.ID, error) {
	if e.parseIDs != nil {
		return e.parseIDs(values)
	}
	return parseIDs(values)
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

	req, err := selectOptionsRequestFromInput(input, authCtx, entry)
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
		gqlmodel.SelectOptionResourceEDIConnection: {
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
		gqlmodel.SelectOptionResourceFuelCard: {
			resolve: r.resolveFuelCardSelectOptions,
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
		gqlmodel.SelectOptionResourceEDITransfer: {
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
		gqlmodel.SelectOptionResourceIFTAFuelType: {
			resolve:  r.resolveIFTAFuelTypeSelectOptions,
			parseIDs: parseCatalogIDs,
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
		gqlmodel.SelectOptionResourceCaptureDevice: {
			resolve: r.resolveCaptureDeviceSelectOptions,
		},
		gqlmodel.SelectOptionResourceCaptureProfile: {
			resolve: r.resolveCaptureProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceDetentionPolicy: {
			resolve: r.resolveDetentionPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceShiftTemplate: {
			resolve: r.resolveShiftTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorkerPolicy: {
			resolve: r.resolveWorkerPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceJobPosition: {
			resolve: r.resolveJobPositionSelectOptions,
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
		gqlmodel.SelectOptionResourceEDICommunicationProfile: {
			resolve: r.resolveEDICommunicationProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIDocumentType: {
			resolve:  r.resolveEDIDocumentTypeSelectOptions,
			parseIDs: parseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEDIMappingProfile: {
			resolve: r.resolveEDIMappingProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIPartner: {
			resolve: r.resolveEDIPartnerSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIPartnerDocumentProfile: {
			resolve: r.resolveEDIPartnerDocumentProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDITemplate: {
			resolve: r.resolveEDITemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDITransactionSet: {
			resolve:  r.resolveEDITransactionSetSelectOptions,
			parseIDs: parseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEmailProfile: {
			resolve: r.resolveEmailProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourcePayCode: {
			resolve: r.resolvePayCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourcePayProfile: {
			resolve: r.resolvePayProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorkerCredentialType: {
			resolve: r.resolveWorkerCredentialTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceTrainingCourse: {
			resolve: r.resolveTrainingCourseSelectOptions,
		},
		gqlmodel.SelectOptionResourcePerformanceReviewTemplate: {
			resolve: r.resolvePerformanceReviewTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourcePTOPolicy: {
			resolve: r.resolvePTOPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceBenefitPlan: {
			resolve: r.resolveBenefitPlanSelectOptions,
		},
		gqlmodel.SelectOptionResourceIFTAJurisdiction: {
			resolve: r.resolveIFTAJurisdictionSelectOptions,
		},
	}
}

func selectOptionsRequestFromInput(
	input gqlmodel.SelectOptionsInput,
	authCtx *authctx.AuthContext,
	entry selectOptionRegistryEntry,
) (*selectOptionsRequest, error) {
	ids, err := entry.ids(input.Ids)
	if err != nil {
		return nil, err
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
	return &selectOptionsRequest{
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

func selectOptionInt16Filter(filters map[string]any, key string) int16 {
	value, ok := filters[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int16(v)
	case int:
		return int16(v) //nolint:gosec // callers pass calendar years
	case int64:
		return int16(v) //nolint:gosec // callers pass calendar years
	case string:
		parsed, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return 0
		}
		return int16(parsed)
	default:
		return 0
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

func ediCommunicationProfileSelectOptionItem(
	entity *edi.EDICommunicationProfile,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediCommunicationProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediCommunicationProfileSelectOption(
	entity *edi.EDICommunicationProfile,
) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(string(entity.Method)),
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
		Description: stringutils.Ptr(string(entity.TransactionSet) + " / " + string(entity.Direction)),
		Meta: map[string]any{
			"code":           entity.Code,
			"transactionSet": string(entity.TransactionSet),
			"direction":      string(entity.Direction),
			"defaultVersion": entity.DefaultVersion,
		},
	}
}

func ediTransactionSetSelectOptionItem(entity *edi.EDITransactionSet) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediTransactionSetSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediTransactionSetSelectOption(entity *edi.EDITransactionSet) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       string(entity.Code) + " - " + entity.Name,
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"code":           string(entity.Code),
			"standard":       string(entity.Standard),
			"defaultVersion": entity.DefaultVersion,
			"status":         string(entity.Status),
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
		Description: stringutils.Ptr(entity.Description),
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
		Description: stringutils.Ptr(string(entity.Kind)),
		Meta: map[string]any{
			"code": entity.Code,
			"kind": string(entity.Kind),
		},
	}
}

func ediPartnerDocumentProfileSelectOptionItem(
	entity *edi.EDIPartnerDocumentProfile,
) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		ediPartnerDocumentProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func ediPartnerDocumentProfileSelectOption(
	entity *edi.EDIPartnerDocumentProfile,
) *gqlmodel.SelectOption {
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
		Description: stringutils.Ptr(description),
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
		Description: stringutils.Ptr(entity.Description),
		Meta: map[string]any{
			"status": string(entity.Status),
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
		Description: stringutils.Ptr(description),
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
		Description: stringutils.Ptr(description),
		Meta: map[string]any{
			"method":                 string(entity.Method),
			"status":                 string(entity.Status),
			"sourceOrganizationId":   entity.SourceOrganizationID.String(),
			"targetOrganizationId":   entity.TargetOrganizationID.String(),
			"sourceOrganizationName": source,
			"targetOrganizationName": target,
			"sourcePartnerId":        optionalIDMeta(entity.SourcePartnerID),
			"targetPartnerId":        optionalIDMeta(entity.TargetPartnerID),
		},
	}
}

func ediConnectionID(entity *edi.EDIConnection) pulid.ID {
	return entity.ID
}

func ediTransferID(entity *edi.EDITransfer) pulid.ID {
	return entity.ID
}

func optionalIDMeta(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}

	return id.String()
}
