package accountingtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	ToolListMappingGaps = "accounting_list_mapping_gaps"
)

const maxMappingReasonRunes = 500

const ParamAccountingSystem = "system"

const paramRecordID = "recordId"

const paramMappingKey = "key"

const ParamMappingID = "mappingId"

const paramTargetType = "targetType"

const toolGetAccountingMapping = "accounting_get_mapping"

type accountingMappingWriter interface {
	FindMapping(
		ctx context.Context,
		req *accountingmappingports.SetAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
	GetReferenceObjects(
		ctx context.Context,
		req *accountingmappingports.GetAccountingReferenceObjectsRequest,
	) ([]*accountingmapping.AccountingReferenceObject, error)
	Set(
		ctx context.Context,
		req *accountingmappingports.SetAccountingMappingRequest,
	) (*accountingmapping.AccountingMapping, error)
	Clear(
		ctx context.Context,
		req *accountingmappingports.AccountingMappingActionRequest,
	) (*accountingmapping.AccountingMapping, error)
	CreateReferenceRecord(
		ctx context.Context,
		req *accountingmappingports.CreateAccountingReferenceRecordRequest,
	) (*accountingmapping.AccountingMapping, error)
	RequestRefresh(
		ctx context.Context,
		req *accountingmappingports.AccountingSetupRequest,
	) (*accountingconnection.AccountingConnection, error)
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*accountingmappingports.AccountingMappingSummary, error)
	GuardHistory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		row *accountingmapping.AccountingMapping,
		acknowledged bool,
	) (int, error)
}

func SystemSchema() map[string]any {
	return agenttoolschema.Enum(
		"The accounting system, such as \"QuickBooksOnline\" or \"Xero\".",
		agenttoolschema.AccountingSystems,
	)
}

func mappingTargetProperties() map[string]any {
	return map[string]any{
		ParamAccountingSystem: SystemSchema(),
		ParamMappingID: agenttoolschema.KindID(
			"The mapping's id from accounting_list_mapping_gaps or "+
				"accounting_get_mapping. Never guess one.",
			recordkind.AccountingMapping,
		),
		paramTargetType: agenttoolschema.Enum(
			"The kind of Trenova record or setting, when naming it by recordId or key.",
			agenttoolschema.MappingTargetTypes,
		),
		paramRecordID: agenttoolschema.KindID(
			"The Trenova customer, carrier or accessorial charge id, from "+
				"customer_list, carrier_list or masterdata_list_accessorial_charges.",
			recordkind.Kind(permission.ResourceCustomer),
			recordkind.Kind(permission.ResourceCarrier),
			recordkind.Kind(permission.ResourceAccessorialCharge),
			recordkind.Kind(permission.ResourceWorker),
			recordkind.Kind(permission.ResourceGeneralLedgerAccount),
		),
		paramMappingKey: jsonschemautils.Text(
			"The setting's key from accounting_list_mapping_gaps, for the " +
				"other target types. Examples: " +
				"\"ARAccount\", \"Freight\", \"Net30\", \"ACH\".",
		),
	}
}

func SystemFrom(params map[string]any) (integration.Type, error) {
	system, err := toolparams.RequireString(params, ParamAccountingSystem)
	if err != nil {
		return "", err
	}
	typ := integration.Type(system)
	if !accountingprovider.SupportsAccountingSync(typ) {
		return "", agenttoolschema.ErrAccountingSystem()
	}
	return typ, nil
}

func mappingRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*accountingmappingports.SetAccountingMappingRequest, error) {
	system, err := SystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	mappingID, _, err := toolparams.OptionalPulid(params.Params, ParamMappingID)
	if err != nil {
		return nil, err
	}
	recordID, _, err := toolparams.OptionalPulid(params.Params, paramRecordID)
	if err != nil {
		return nil, err
	}
	req := &accountingmappingports.SetAccountingMappingRequest{
		TenantInfo:      toolparams.TenantFrom(*params),
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		MappingID:       mappingID,
		TargetType: accountingsync.MappingTargetType(
			toolparams.OptionalString(params.Params, paramTargetType),
		),
		TrenovaObjectID: recordID,
		TrenovaKey:      toolparams.OptionalString(params.Params, paramMappingKey),
		Source:          accountingmapping.MappingSourceAgent,
	}
	if err = req.ValidateTarget(); err != nil {
		return nil, err
	}
	return req, nil
}

func findMapping(
	ctx context.Context,
	tool agenttoolports.AgentTool,
	mappings accountingMappingWriter,
	params *agenttoolports.ToolExecuteParams,
) (*accountingmappingports.SetAccountingMappingRequest, *accountingmapping.AccountingMapping, error) {
	if err := toolguard.GuardExecute(tool, *params); err != nil {
		return nil, nil, err
	}
	req, err := mappingRequestFrom(params)
	if err != nil {
		return nil, nil, err
	}
	row, err := mappings.FindMapping(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return req, row, nil
}

func MappedLabel(row *accountingmapping.AccountingMapping) string {
	if row.ExternalName == "" {
		return "nothing"
	}
	return row.ExternalName
}

func accountingMappingPolicy(
	name string,
	reversible bool,
	rationale string,
) agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          name,
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    reversible,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     rationale,
	}
}

type SetAccountingMappingTool struct {
	mappings accountingMappingWriter
}

func newSetAccountingMappingTool(mappings accountingMappingWriter) agenttoolports.AgentTool {
	return &SetAccountingMappingTool{mappings: mappings}
}

func ProvideSetAccountingMappingTool(
	mappings accountingmappingports.AccountingMappingService,
) agenttoolports.AgentTool {
	return newSetAccountingMappingTool(mappings)
}

func (t *SetAccountingMappingTool) Name() string { return "accounting_set_mapping" }

func (t *SetAccountingMappingTool) Recipe() []string {
	return []string{
		ToolListMappingGaps,
		"accounting_get_mapping",
		"accounting_set_mapping",
	}
}

func (t *SetAccountingMappingTool) Description() string {
	return "Choose which accounting system record a Trenova record or setting is sent as, " +
		"and confirm it. Use an externalId from the mapping's candidates or from " +
		"the options accounting_get_mapping finds with a query; never guess one. To accept Trenova's " +
		"proposal, send the proposed record's externalId. Give a short reason a bookkeeper " +
		"would accept. Only confirmed mappings are used when syncing."
}

func (t *SetAccountingMappingTool) SearchTerms() []string {
	return []string{"map", "match", "item", "account", "confirm", "link"}
}

func (t *SetAccountingMappingTool) Prerequisites() []string {
	return []string{ToolListMappingGaps, toolGetAccountingMapping}
}

func (t *SetAccountingMappingTool) ParamSchema() map[string]any {
	properties := mappingTargetProperties()
	properties["externalId"] = jsonschemautils.Text(
		"The accounting system record's id, from accounting_get_mapping's " +
			"candidates or search options.",
	)
	properties["reason"] = jsonschemautils.Text(
		"Why this record is the right one, in one sentence.",
	)
	return jsonschemautils.Object(properties, ParamAccountingSystem, "externalId", "reason")
}

func (t *SetAccountingMappingTool) Policy() agenttoolports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), true,
		"Decides which account or record Trenova's invoices, payments and bills will post "+
			"to, so a person approves it; clearing or changing it undoes it.")
}

func (t *SetAccountingMappingTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingmappingports.SetAccountingMappingRequest, *accountingmapping.AccountingMapping, *accountingmapping.AccountingReferenceObject, error) {
	req, row, err := findMapping(ctx, t, t.mappings, params)
	if err != nil {
		return nil, nil, nil, err
	}
	externalID, err := toolparams.RequireString(params.Params, "externalId")
	if err != nil {
		return nil, nil, nil, err
	}
	reason, err := toolparams.RequireString(params.Params, "reason")
	if err != nil {
		return nil, nil, nil, err
	}
	refs, err := t.mappings.GetReferenceObjects(
		ctx,
		&accountingmappingports.GetAccountingReferenceObjectsRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: row.ConnectionID,
			Kind:         row.ProviderKind,
			ExternalIDs:  []string{externalID},
		},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(refs) == 0 || !refs[0].Usable() {
		return nil, nil, nil, errortypes.NewValidationError(
			"externalId",
			errortypes.ErrInvalid,
			"{0} is not a usable {1} in the accounting system; pick one from the candidates or search options",
			externalID,
			string(row.ProviderKind),
		)
	}

	req.MappingID = row.ID
	req.ExternalID = externalID
	req.Reason = stringutils.TruncateRunes(strings.TrimSpace(reason), maxMappingReasonRunes)
	return req, row, refs[0], nil
}

func (t *SetAccountingMappingTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, _, err := t.request(ctx, &params)
	return err
}

func (t *SetAccountingMappingTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.Set(ctx, req)
	return err
}

type ClearAccountingMappingTool struct {
	mappings accountingMappingWriter
}

func newClearAccountingMappingTool(mappings accountingMappingWriter) agenttoolports.AgentTool {
	return &ClearAccountingMappingTool{mappings: mappings}
}

func ProvideClearAccountingMappingTool(
	mappings accountingmappingports.AccountingMappingService,
) agenttoolports.AgentTool {
	return newClearAccountingMappingTool(mappings)
}

func (t *ClearAccountingMappingTool) Name() string { return "accounting_clear_mapping" }

func (t *ClearAccountingMappingTool) Description() string {
	return "Remove the accounting system record a Trenova record or setting is mapped to, " +
		"so it is unmatched again and that record is never proposed for it again. Use it " +
		"when a person says a mapping is wrong and no right record exists yet; to replace it " +
		"with another record, use accounting_set_mapping instead."
}

func (t *ClearAccountingMappingTool) SearchTerms() []string {
	return []string{"unmap", "remove", "wrong", "mapping"}
}

func (t *ClearAccountingMappingTool) Prerequisites() []string {
	return []string{toolGetAccountingMapping}
}

func (t *ClearAccountingMappingTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(mappingTargetProperties(), ParamAccountingSystem)
}

func (t *ClearAccountingMappingTool) Policy() agenttoolports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), true,
		"Unmatches a record so it cannot sync until someone maps it again; mapping it "+
			"again undoes it.")
}

func (t *ClearAccountingMappingTool) mapping(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingmapping.AccountingMapping, error) {
	_, row, err := findMapping(ctx, t, t.mappings, params)
	if err != nil {
		return nil, err
	}
	if row.State == accountingmapping.MappingStateUnmatched {
		return nil, errortypes.NewBusinessError("{0} is not mapped", row.TargetLabel)
	}
	return row, nil
}

func (t *ClearAccountingMappingTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.mapping(ctx, &params)
	return err
}

func (t *ClearAccountingMappingTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.Clear(ctx, &accountingmappingports.AccountingMappingActionRequest{
		TenantInfo: toolparams.TenantFrom(params),
		UserID:     params.Actor.UserID,
		ID:         row.ID,
		Source:     accountingmapping.MappingSourceAgent,
	})
	return err
}

type CreateAccountingReferenceRecordTool struct {
	mappings accountingMappingWriter
}

func newCreateAccountingReferenceRecordTool(
	mappings accountingMappingWriter,
) agenttoolports.AgentTool {
	return &CreateAccountingReferenceRecordTool{mappings: mappings}
}

func ProvideCreateAccountingReferenceRecordTool(
	mappings accountingmappingports.AccountingMappingService,
) agenttoolports.AgentTool {
	return newCreateAccountingReferenceRecordTool(mappings)
}

func (t *CreateAccountingReferenceRecordTool) Name() string {
	return "accounting_create_reference_record"
}

func (t *CreateAccountingReferenceRecordTool) Recipe() []string {
	return []string{"accounting_get_mapping", "accounting_create_reference_record"}
}

func (t *CreateAccountingReferenceRecordTool) Description() string {
	return "Create an item, customer or vendor in the accounting system and map a Trenova " +
		"record to it. Use it for an accessorial charge, line type, customer or carrier " +
		"with no match there, and only when accounting_get_mapping finds no " +
		"existing record; a duplicate is refused and names the existing one. Accounts, " +
		"terms and payment methods are the bookkeeper's and are never created."
}

func (t *CreateAccountingReferenceRecordTool) SearchTerms() []string {
	return []string{"create", "item", "vendor", "add"}
}

func (t *CreateAccountingReferenceRecordTool) Prerequisites() []string {
	return []string{toolGetAccountingMapping}
}

func (t *CreateAccountingReferenceRecordTool) ParamSchema() map[string]any {
	properties := mappingTargetProperties()
	properties["name"] = jsonschemautils.Text(
		"The name to create it under. Defaults to the Trenova record's name.",
	)
	return jsonschemautils.Object(properties, ParamAccountingSystem)
}

func (t *CreateAccountingReferenceRecordTool) Policy() agenttoolports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), false,
		"Creates a record in the organization's own accounting system; Trenova cannot "+
			"delete it again, so a person approves it.")
}

func (t *CreateAccountingReferenceRecordTool) mapping(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingmapping.AccountingMapping, error) {
	_, row, err := findMapping(ctx, t, t.mappings, params)
	if err != nil {
		return nil, err
	}
	if !row.ProviderKind.Creatable() {
		return nil, errortypes.NewBusinessError(
			"{0} records are kept by the bookkeeper and are not created from Trenova",
			string(row.ProviderKind),
		)
	}
	if row.State == accountingmapping.MappingStateConfirmed {
		return nil, errortypes.NewBusinessError(
			"{0} is already mapped to {1}",
			row.TargetLabel,
			row.ExternalName,
		)
	}
	return row, nil
}

func (t *CreateAccountingReferenceRecordTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.mapping(ctx, &params)
	return err
}

func (t *CreateAccountingReferenceRecordTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.CreateReferenceRecord(
		ctx,
		&accountingmappingports.CreateAccountingReferenceRecordRequest{
			TenantInfo: toolparams.TenantFrom(params),
			UserID:     params.Actor.UserID,
			MappingID:  row.ID,
			Name:       strings.TrimSpace(toolparams.OptionalString(params.Params, "name")),
			Source:     accountingmapping.MappingSourceAgent,
		},
	)
	return err
}

type RefreshAccountingReferenceDataTool struct {
	mappings accountingMappingWriter
}

func newRefreshAccountingReferenceDataTool(
	mappings accountingMappingWriter,
) agenttoolports.AgentTool {
	return &RefreshAccountingReferenceDataTool{mappings: mappings}
}

func ProvideRefreshAccountingReferenceDataTool(
	mappings accountingmappingports.AccountingMappingService,
) agenttoolports.AgentTool {
	return newRefreshAccountingReferenceDataTool(mappings)
}

func (t *RefreshAccountingReferenceDataTool) Name() string {
	return "accounting_refresh_reference_data"
}

func (t *RefreshAccountingReferenceDataTool) Recipe() []string {
	return []string{"accounting_refresh_reference_data", ToolListMappingGaps}
}

func (t *RefreshAccountingReferenceDataTool) Description() string {
	return "Read the accounting system's accounts, items, customers, vendors, terms and " +
		"payment methods again and refresh Trenova's proposals. Use it after a person says " +
		"they added or renamed something in the accounting system. It runs in the " +
		"background; check accounting_list_mapping_gaps afterwards. It changes nothing in " +
		"the books and never touches a confirmed mapping."
}

func (t *RefreshAccountingReferenceDataTool) SearchTerms() []string {
	return []string{"pull", "reload", "sync", "new accounts", "refresh"}
}

func (t *RefreshAccountingReferenceDataTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(
		map[string]any{ParamAccountingSystem: SystemSchema()},
		ParamAccountingSystem,
	)
}

func (t *RefreshAccountingReferenceDataTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    false,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Reads the accounting system and refreshes Trenova's suggestions; it " +
			"writes nothing to the books and leaves confirmed mappings alone.",
	}
}

func (t *RefreshAccountingReferenceDataTool) setup(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingmappingports.AccountingSetupRequest, *accountingmappingports.AccountingMappingSummary, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	system, err := SystemFrom(params.Params)
	if err != nil {
		return nil, nil, err
	}
	req := &accountingmappingports.AccountingSetupRequest{
		TenantInfo:      toolparams.TenantFrom(*params),
		UserID:          params.Actor.UserID,
		IntegrationType: system,
	}
	summary, err := t.mappings.Summary(ctx, req.TenantInfo, system)
	if err != nil {
		return nil, nil, err
	}
	if summary.Connection == nil || !summary.Connection.IsActive() {
		return nil, nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			summary.ProviderName,
		)
	}
	return req, summary, nil
}

func (t *RefreshAccountingReferenceDataTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.setup(ctx, &params)
	return err
}

func (t *RefreshAccountingReferenceDataTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.setup(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.RequestRefresh(ctx, req)
	return err
}
