package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxMappingReasonRunes    = 500
	paramAccountingSystem    = "system"
	paramRecordID            = "recordId"
	paramMappingKey          = "key"
	paramMappingID           = "mappingId"
	paramTargetType          = "targetType"
	toolGetAccountingMapping = "get_accounting_mapping"
	fieldMappedTo            = "mappedTo"
	fieldMappingState        = "state"
)

type accountingMappingWriter interface {
	FindMapping(
		ctx context.Context,
		req *serviceports.SetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	GetReferenceObjects(
		ctx context.Context,
		req *serviceports.GetAccountingReferenceObjectsRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
	Set(
		ctx context.Context,
		req *serviceports.SetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	Clear(
		ctx context.Context,
		req *serviceports.AccountingMappingActionRequest,
	) (*accountingsync.AccountingMapping, error)
	CreateReferenceRecord(
		ctx context.Context,
		req *serviceports.CreateAccountingReferenceRecordRequest,
	) (*accountingsync.AccountingMapping, error)
	RequestRefresh(
		ctx context.Context,
		req *serviceports.AccountingSetupRequest,
	) (*accountingsync.AccountingConnection, error)
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*serviceports.AccountingMappingSummary, error)
}

func accountingSystemSchema() map[string]any {
	return jsonschemautils.Enum(
		"The accounting system. Example: \"QuickBooksOnline\".",
		string(integration.TypeQuickBooksOnline),
	)
}

func mappingTargetProperties() map[string]any {
	types := accountingsync.AllMappingTargetTypes()
	values := make([]string, 0, len(types))
	for _, targetType := range types {
		values = append(values, string(targetType))
	}
	return map[string]any{
		paramAccountingSystem: accountingSystemSchema(),
		paramMappingID: jsonschemautils.Text(
			"The mapping's id from list_accounting_mapping_gaps or " +
				"get_accounting_mapping. Never guess one.",
		),
		paramTargetType: jsonschemautils.Enum(
			"The kind of Trenova record or setting, when naming it by recordId or key.",
			values...,
		),
		paramRecordID: jsonschemautils.Text(
			"The Trenova customer, carrier or accessorial charge id, from " +
				"list_customers, list_carriers or list_accessorial_charges.",
		),
		paramMappingKey: jsonschemautils.Text(
			"The setting's key from list_accounting_mapping_gaps, for the " +
				"other target types. Examples: " +
				"\"ARAccount\", \"Freight\", \"Net30\", \"ACH\".",
		),
	}
}

func accountingSystemFrom(params map[string]any) (integration.Type, error) {
	system, err := requireString(params, paramAccountingSystem)
	if err != nil {
		return "", err
	}
	typ := integration.Type(system)
	if !accountingsync.SupportsAccountingSync(typ) {
		return "", fmt.Errorf("system must be %q", integration.TypeQuickBooksOnline)
	}
	return typ, nil
}

func mappingRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*serviceports.SetAccountingMappingRequest, error) {
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, err
	}
	mappingID, _, err := optionalPulid(params.Params, paramMappingID)
	if err != nil {
		return nil, err
	}
	recordID, _, err := optionalPulid(params.Params, paramRecordID)
	if err != nil {
		return nil, err
	}
	req := &serviceports.SetAccountingMappingRequest{
		TenantInfo:      tenantFrom(*params),
		UserID:          params.Actor.UserID,
		IntegrationType: system,
		MappingID:       mappingID,
		TargetType: accountingsync.MappingTargetType(
			optionalString(params.Params, paramTargetType),
		),
		TrenovaObjectID: recordID,
		TrenovaKey:      optionalString(params.Params, paramMappingKey),
		Source:          accountingsync.MappingSourceAgent,
	}
	if err = req.ValidateTarget(); err != nil {
		return nil, err
	}
	return req, nil
}

func findMapping(
	ctx context.Context,
	tool serviceports.AgentTool,
	mappings accountingMappingWriter,
	params *serviceports.ToolExecuteParams,
) (*serviceports.SetAccountingMappingRequest, *accountingsync.AccountingMapping, error) {
	if err := guardExecute(tool, *params); err != nil {
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

func mappedLabel(row *accountingsync.AccountingMapping) string {
	if row.ExternalName == "" {
		return "nothing"
	}
	return row.ExternalName
}

func accountingMappingPolicy(
	name string,
	reversible bool,
	rationale string,
) serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          name,
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    reversible,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     rationale,
	}
}

type setAccountingMappingTool struct {
	mappings accountingMappingWriter
}

func newSetAccountingMappingTool(mappings accountingMappingWriter) serviceports.AgentTool {
	return &setAccountingMappingTool{mappings: mappings}
}

func provideSetAccountingMappingTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentTool {
	return newSetAccountingMappingTool(mappings)
}

func (t *setAccountingMappingTool) Name() string { return "set_accounting_mapping" }

func (t *setAccountingMappingTool) Description() string {
	return "Choose which accounting system record a Trenova record or setting is sent as, " +
		"and confirm it. Use an externalId from the mapping's candidates or from " +
		"get_accounting_mapping's search options; never guess one. To accept Trenova's " +
		"proposal, send the proposed record's externalId. Give a short reason a bookkeeper " +
		"would accept. Only confirmed mappings are used when syncing."
}

func (t *setAccountingMappingTool) SearchTerms() []string {
	return []string{"map", "match", "item", "account", "confirm", "link"}
}

func (t *setAccountingMappingTool) Prerequisites() []string {
	return []string{"list_accounting_mapping_gaps", toolGetAccountingMapping}
}

func (t *setAccountingMappingTool) ParamSchema() map[string]any {
	properties := mappingTargetProperties()
	properties["externalId"] = jsonschemautils.Text(
		"The accounting system record's id, from get_accounting_mapping's " +
			"candidates or search options.",
	)
	properties["reason"] = jsonschemautils.Text(
		"Why this record is the right one, in one sentence.",
	)
	return jsonschemautils.Object(properties, paramAccountingSystem, "externalId", "reason")
}

func (t *setAccountingMappingTool) Policy() serviceports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), true,
		"Decides which account or record Trenova's invoices, payments and bills will post "+
			"to, so a person approves it; clearing or changing it undoes it.")
}

func (t *setAccountingMappingTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.SetAccountingMappingRequest, *accountingsync.AccountingMapping, *accountingsync.AccountingReferenceObject, error) {
	req, row, err := findMapping(ctx, t, t.mappings, params)
	if err != nil {
		return nil, nil, nil, err
	}
	externalID, err := requireString(params.Params, "externalId")
	if err != nil {
		return nil, nil, nil, err
	}
	reason, err := requireString(params.Params, "reason")
	if err != nil {
		return nil, nil, nil, err
	}
	refs, err := t.mappings.GetReferenceObjects(
		ctx,
		&serviceports.GetAccountingReferenceObjectsRequest{
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

func (t *setAccountingMappingTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, _, err := t.request(ctx, &params)
	return err
}

func (t *setAccountingMappingTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.Set(ctx, req)
	return err
}

func (t *setAccountingMappingTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	_, row, ref, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would send %s as %s in the accounting system and confirm it.",
			row.TargetLabel,
			ref.Label(),
		),
		Previewed: true,
		Changes: []agent.FieldChange{
			{Field: fieldMappedTo, From: mappedLabel(row), To: ref.Label()},
			{
				Field: fieldMappingState,
				From:  string(row.State),
				To:    string(accountingsync.MappingStateConfirmed),
			},
		},
	}, nil
}

type clearAccountingMappingTool struct {
	mappings accountingMappingWriter
}

func newClearAccountingMappingTool(mappings accountingMappingWriter) serviceports.AgentTool {
	return &clearAccountingMappingTool{mappings: mappings}
}

func provideClearAccountingMappingTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentTool {
	return newClearAccountingMappingTool(mappings)
}

func (t *clearAccountingMappingTool) Name() string { return "clear_accounting_mapping" }

func (t *clearAccountingMappingTool) Description() string {
	return "Remove the accounting system record a Trenova record or setting is mapped to, " +
		"so it is unmatched again and that record is never proposed for it again. Use it " +
		"when a person says a mapping is wrong and no right record exists yet; to replace it " +
		"with another record, use set_accounting_mapping instead."
}

func (t *clearAccountingMappingTool) SearchTerms() []string {
	return []string{"unmap", "remove", "wrong", "mapping"}
}

func (t *clearAccountingMappingTool) Prerequisites() []string {
	return []string{toolGetAccountingMapping}
}

func (t *clearAccountingMappingTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(mappingTargetProperties(), paramAccountingSystem)
}

func (t *clearAccountingMappingTool) Policy() serviceports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), true,
		"Unmatches a record so it cannot sync until someone maps it again; mapping it "+
			"again undoes it.")
}

func (t *clearAccountingMappingTool) mapping(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*accountingsync.AccountingMapping, error) {
	_, row, err := findMapping(ctx, t, t.mappings, params)
	if err != nil {
		return nil, err
	}
	if row.State == accountingsync.MappingStateUnmatched {
		return nil, errortypes.NewBusinessError("{0} is not mapped", row.TargetLabel)
	}
	return row, nil
}

func (t *clearAccountingMappingTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.mapping(ctx, &params)
	return err
}

func (t *clearAccountingMappingTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.Clear(ctx, &serviceports.AccountingMappingActionRequest{
		TenantInfo: tenantFrom(params),
		UserID:     params.Actor.UserID,
		ID:         row.ID,
		Source:     accountingsync.MappingSourceAgent,
	})
	return err
}

func (t *clearAccountingMappingTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &agent.ToolSimulation{
		Summary:   fmt.Sprintf("Would unmatch %s from %s.", row.TargetLabel, mappedLabel(row)),
		Previewed: true,
		Changes: []agent.FieldChange{
			{Field: fieldMappedTo, From: mappedLabel(row), To: "nothing"},
			{
				Field: fieldMappingState,
				From:  string(row.State),
				To:    string(accountingsync.MappingStateUnmatched),
			},
		},
	}, nil
}

type createAccountingReferenceRecordTool struct {
	mappings accountingMappingWriter
}

func newCreateAccountingReferenceRecordTool(
	mappings accountingMappingWriter,
) serviceports.AgentTool {
	return &createAccountingReferenceRecordTool{mappings: mappings}
}

func provideCreateAccountingReferenceRecordTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentTool {
	return newCreateAccountingReferenceRecordTool(mappings)
}

func (t *createAccountingReferenceRecordTool) Name() string {
	return "create_accounting_reference_record"
}

func (t *createAccountingReferenceRecordTool) Description() string {
	return "Create an item, customer or vendor in the accounting system and map a Trenova " +
		"record to it. Use it for an accessorial charge, line type, customer or carrier " +
		"with no match there, and only when get_accounting_mapping's search finds no " +
		"existing record; a duplicate is refused and names the existing one. Accounts, " +
		"terms and payment methods are the bookkeeper's and are never created."
}

func (t *createAccountingReferenceRecordTool) SearchTerms() []string {
	return []string{"create", "item", "vendor", "add"}
}

func (t *createAccountingReferenceRecordTool) Prerequisites() []string {
	return []string{toolGetAccountingMapping}
}

func (t *createAccountingReferenceRecordTool) ParamSchema() map[string]any {
	properties := mappingTargetProperties()
	properties["name"] = jsonschemautils.Text(
		"The name to create it under. Defaults to the Trenova record's name.",
	)
	return jsonschemautils.Object(properties, paramAccountingSystem)
}

func (t *createAccountingReferenceRecordTool) Policy() serviceports.ToolPolicy {
	return accountingMappingPolicy(t.Name(), false,
		"Creates a record in the organization's own accounting system; Trenova cannot "+
			"delete it again, so a person approves it.")
}

func (t *createAccountingReferenceRecordTool) mapping(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*accountingsync.AccountingMapping, error) {
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
	if row.State == accountingsync.MappingStateConfirmed {
		return nil, errortypes.NewBusinessError(
			"{0} is already mapped to {1}",
			row.TargetLabel,
			row.ExternalName,
		)
	}
	return row, nil
}

func (t *createAccountingReferenceRecordTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.mapping(ctx, &params)
	return err
}

func (t *createAccountingReferenceRecordTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.CreateReferenceRecord(
		ctx,
		&serviceports.CreateAccountingReferenceRecordRequest{
			TenantInfo: tenantFrom(params),
			UserID:     params.Actor.UserID,
			MappingID:  row.ID,
			Name:       strings.TrimSpace(optionalString(params.Params, "name")),
			Source:     accountingsync.MappingSourceAgent,
		},
	)
	return err
}

func (t *createAccountingReferenceRecordTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	row, err := t.mapping(ctx, &params)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(optionalString(params.Params, "name"))
	if name == "" {
		name = row.TargetLabel
	}
	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would create the %s %q in the accounting system and map %s to it.",
			strings.ToLower(string(row.ProviderKind)),
			name,
			row.TargetLabel,
		),
		Previewed: true,
		Changes: []agent.FieldChange{
			{Field: fieldMappedTo, From: mappedLabel(row), To: name},
			{
				Field: fieldMappingState,
				From:  string(row.State),
				To:    string(accountingsync.MappingStateConfirmed),
			},
		},
	}, nil
}

type refreshAccountingReferenceDataTool struct {
	mappings accountingMappingWriter
}

func newRefreshAccountingReferenceDataTool(
	mappings accountingMappingWriter,
) serviceports.AgentTool {
	return &refreshAccountingReferenceDataTool{mappings: mappings}
}

func provideRefreshAccountingReferenceDataTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentTool {
	return newRefreshAccountingReferenceDataTool(mappings)
}

func (t *refreshAccountingReferenceDataTool) Name() string {
	return "refresh_accounting_reference_data"
}

func (t *refreshAccountingReferenceDataTool) Description() string {
	return "Read the accounting system's accounts, items, customers, vendors, terms and " +
		"payment methods again and refresh Trenova's proposals. Use it after a person says " +
		"they added or renamed something in the accounting system. It runs in the " +
		"background; check list_accounting_mapping_gaps afterwards. It changes nothing in " +
		"the books and never touches a confirmed mapping."
}

func (t *refreshAccountingReferenceDataTool) SearchTerms() []string {
	return []string{"pull", "reload", "sync", "new accounts", "refresh"}
}

func (t *refreshAccountingReferenceDataTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(
		map[string]any{paramAccountingSystem: accountingSystemSchema()},
		paramAccountingSystem,
	)
}

func (t *refreshAccountingReferenceDataTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAccountingIntegration,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    false,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Reads the accounting system and refreshes Trenova's suggestions; it " +
			"writes nothing to the books and leaves confirmed mappings alone.",
	}
}

func (t *refreshAccountingReferenceDataTool) setup(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.AccountingSetupRequest, *serviceports.AccountingMappingSummary, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return nil, nil, err
	}
	req := &serviceports.AccountingSetupRequest{
		TenantInfo:      tenantFrom(*params),
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

func (t *refreshAccountingReferenceDataTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.setup(ctx, &params)
	return err
}

func (t *refreshAccountingReferenceDataTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.setup(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.mappings.RequestRefresh(ctx, req)
	return err
}

func (t *refreshAccountingReferenceDataTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	_, summary, err := t.setup(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would read %s's records for %s again and refresh the open proposals.",
			summary.ProviderName,
			summary.Connection.ExternalCompanyName,
		),
		Previewed: true,
	}, nil
}
