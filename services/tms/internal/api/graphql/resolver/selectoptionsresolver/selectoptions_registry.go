package selectoptionsresolver

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"

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

type SelectOptionRegistryEntry struct {
	Resolve  func(context.Context, *SelectOptionsRequest) (*gqlmodel.SelectOptionConnection, error)
	parseIDs func([]string) ([]pulid.ID, error)
}

func (e SelectOptionRegistryEntry) IDs(values []string) ([]pulid.ID, error) {
	if e.parseIDs != nil {
		return e.parseIDs(values)
	}
	return base.ParseIDs(values)
}

type SelectOptionsRequest struct {
	TenantInfo  pagination.TenantInfo
	SelectQuery *pagination.SelectQueryRequest
	IDs         []pulid.ID
	Filters     map[string]any
}

type selectOptionConnectionItem struct {
	option *gqlmodel.SelectOption
	cursor pagination.Cursor
}

func (r *Deps) RequireAuthContext(ctx context.Context) (*authctx.AuthContext, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	return authCtx, nil
}

//nolint:funlen // one table entry per select-option resource
func (r *Deps) SelectOptionRegistry() map[gqlmodel.SelectOptionResource]SelectOptionRegistryEntry {
	return map[gqlmodel.SelectOptionResource]SelectOptionRegistryEntry{
		gqlmodel.SelectOptionResourceCarrier: {
			Resolve: r.resolveCarrierSelectOptions,
		},
		gqlmodel.SelectOptionResourceCustomer: {
			Resolve: r.resolveCustomerSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIConnection: {
			Resolve: r.resolveEDIConnectionSelectOptions,
		},
		gqlmodel.SelectOptionResourceEquipmentType: {
			Resolve: r.resolveEquipmentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceEquipmentManufacturer: {
			Resolve: r.resolveEquipmentManufacturerSelectOptions,
		},
		gqlmodel.SelectOptionResourceTrailer: {
			Resolve: r.resolveTrailerSelectOptions,
		},
		gqlmodel.SelectOptionResourceTractor: {
			Resolve: r.resolveTractorSelectOptions,
		},
		gqlmodel.SelectOptionResourceFuelCard: {
			Resolve: r.resolveFuelCardSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorker: {
			Resolve: r.resolveWorkerSelectOptions,
		},
		gqlmodel.SelectOptionResourceUsState: {
			Resolve: r.resolveUSStateSelectOptions,
		},
		gqlmodel.SelectOptionResourceShipment: {
			Resolve: r.resolveShipmentSelectOptions,
		},
		gqlmodel.SelectOptionResourceOrder: {
			Resolve: r.resolveOrderSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDITransfer: {
			Resolve: r.resolveEDITransferSelectOptions,
		},
		gqlmodel.SelectOptionResourceFuelIndex: {
			Resolve: r.resolveFuelIndexSelectOptions,
		},
		gqlmodel.SelectOptionResourceFuelSurchargeProgram: {
			Resolve: r.resolveFuelSurchargeProgramSelectOptions,
		},
		gqlmodel.SelectOptionResourceFiscalYear: {
			Resolve: r.resolveFiscalYearSelectOptions,
		},
		gqlmodel.SelectOptionResourceFiscalPeriod: {
			Resolve: r.resolveFiscalPeriodSelectOptions,
		},
		gqlmodel.SelectOptionResourceGlAccount: {
			Resolve: r.resolveGLAccountSelectOptions,
		},
		gqlmodel.SelectOptionResourceIFTAFuelType: {
			Resolve:  r.resolveIFTAFuelTypeSelectOptions,
			parseIDs: base.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceLocation: {
			Resolve: r.resolveLocationSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateZone: {
			Resolve: r.resolveRateZoneSelectOptions,
		},
		gqlmodel.SelectOptionResourceFleetCode: {
			Resolve: r.resolveFleetCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourceShipmentType: {
			Resolve: r.resolveShipmentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceServiceType: {
			Resolve: r.resolveServiceTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceLocationCategory: {
			Resolve: r.resolveLocationCategorySelectOptions,
		},
		gqlmodel.SelectOptionResourceDistanceProfile: {
			Resolve: r.resolveDistanceProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceOrganization: {
			Resolve: r.resolveOrganizationSelectOptions,
		},
		gqlmodel.SelectOptionResourceUser: {
			Resolve: r.resolveUserSelectOptions,
		},
		gqlmodel.SelectOptionResourceRole: {
			Resolve: r.resolveRoleSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateMatrix: {
			Resolve: r.resolveRateMatrixSelectOptions,
		},
		gqlmodel.SelectOptionResourceRateAgreement: {
			Resolve: r.resolveRateAgreementSelectOptions,
		},
		gqlmodel.SelectOptionResourceAccessorialCharge: {
			Resolve: r.resolveAccessorialChargeSelectOptions,
		},
		gqlmodel.SelectOptionResourceAccountType: {
			Resolve: r.resolveAccountTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceCommodity: {
			Resolve: r.resolveCommoditySelectOptions,
		},
		gqlmodel.SelectOptionResourceDocumentType: {
			Resolve: r.resolveDocumentTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceCaptureDevice: {
			Resolve: r.resolveCaptureDeviceSelectOptions,
		},
		gqlmodel.SelectOptionResourceCaptureProfile: {
			Resolve: r.resolveCaptureProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceDetentionPolicy: {
			Resolve: r.resolveDetentionPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceShiftTemplate: {
			Resolve: r.resolveShiftTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorkerPolicy: {
			Resolve: r.resolveWorkerPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceJobPosition: {
			Resolve: r.resolveJobPositionSelectOptions,
		},
		gqlmodel.SelectOptionResourceFormulaTemplate: {
			Resolve: r.resolveFormulaTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceHazardousMaterial: {
			Resolve: r.resolveHazardousMaterialSelectOptions,
		},
		gqlmodel.SelectOptionResourceServiceFailureReasonCode: {
			Resolve: r.resolveServiceFailureReasonCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDICommunicationProfile: {
			Resolve: r.resolveEDICommunicationProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIDocumentType: {
			Resolve:  r.resolveEDIDocumentTypeSelectOptions,
			parseIDs: base.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEDIMappingProfile: {
			Resolve: r.resolveEDIMappingProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIPartner: {
			Resolve: r.resolveEDIPartnerSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDIPartnerDocumentProfile: {
			Resolve: r.resolveEDIPartnerDocumentProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDITemplate: {
			Resolve: r.resolveEDITemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourceEDITransactionSet: {
			Resolve:  r.resolveEDITransactionSetSelectOptions,
			parseIDs: base.ParseCatalogIDs,
		},
		gqlmodel.SelectOptionResourceEmailProfile: {
			Resolve: r.resolveEmailProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourcePayCode: {
			Resolve: r.resolvePayCodeSelectOptions,
		},
		gqlmodel.SelectOptionResourcePayProfile: {
			Resolve: r.resolvePayProfileSelectOptions,
		},
		gqlmodel.SelectOptionResourceWorkerCredentialType: {
			Resolve: r.resolveWorkerCredentialTypeSelectOptions,
		},
		gqlmodel.SelectOptionResourceTrainingCourse: {
			Resolve: r.resolveTrainingCourseSelectOptions,
		},
		gqlmodel.SelectOptionResourcePerformanceReviewTemplate: {
			Resolve: r.resolvePerformanceReviewTemplateSelectOptions,
		},
		gqlmodel.SelectOptionResourcePTOPolicy: {
			Resolve: r.resolvePTOPolicySelectOptions,
		},
		gqlmodel.SelectOptionResourceBenefitPlan: {
			Resolve: r.resolveBenefitPlanSelectOptions,
		},
		gqlmodel.SelectOptionResourceIFTAJurisdiction: {
			Resolve: r.resolveIFTAJurisdictionSelectOptions,
		},
	}
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
		PageInfo: base.PageInfo(
			hasNextPage,
			base.LastEdgeCursor(edges, func(edge *gqlmodel.SelectOptionEdge) string {
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
		ID:    entity.ID.String(),
		Label: entity.Code + " - " + entity.Name,
		Description: stringutils.Ptr(
			string(entity.TransactionSet) + " / " + string(entity.Direction),
		),
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
