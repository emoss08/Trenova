package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	paramAccountingSystem = "system"
	paramRecordID         = "recordId"
	paramMappingKey       = "key"
	paramMappingID        = "mappingId"
	paramTargetType       = "targetType"

	mappingGapsDefaultLimit = 25
	mappingGapsMaxLimit     = 50
	mappingOptionsLimit     = 10
)

type accountingMappingReader interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*serviceports.AccountingMappingSummary, error)
	ListMappings(
		ctx context.Context,
		req *serviceports.ListAccountingMappingsRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error)
	FindMapping(
		ctx context.Context,
		req *serviceports.SetAccountingMappingRequest,
	) (*accountingsync.AccountingMapping, error)
	SearchReference(
		ctx context.Context,
		req *serviceports.SearchAccountingReferenceRequest,
	) ([]*accountingsync.AccountingReferenceObject, error)
}

type accountingRecordRow struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Number     string `json:"number,omitempty"`
}

type accountingCandidateRow struct {
	ExternalID string  `json:"externalId"`
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason,omitempty"`
}

type accountingMappingRow struct {
	ID          string                   `json:"id"`
	TargetType  string                   `json:"targetType"`
	Target      string                   `json:"target"`
	RecordID    string                   `json:"recordId,omitempty"`
	Key         string                   `json:"key,omitempty"`
	RecordKind  string                   `json:"recordKind"`
	State       string                   `json:"state"`
	Required    bool                     `json:"required"`
	MappedTo    *accountingRecordRow     `json:"mappedTo,omitempty"`
	Source      string                   `json:"source,omitempty"`
	Confidence  *float64                 `json:"confidence,omitempty"`
	Reason      string                   `json:"reason,omitempty"`
	Candidates  []accountingCandidateRow `json:"candidates,omitempty"`
	ConfirmedOn optionalDate             `json:"confirmedOn"`
	CanCreate   bool                     `json:"canCreateInAccountingSystem"`
}

type accountingMappingGroupRow struct {
	TargetType string `json:"targetType"`
	Unmatched  int    `json:"unmatched"`
	Proposed   int    `json:"proposed"`
	Confirmed  int    `json:"confirmed"`
}

type accountingMappingGapsResult struct {
	Provider          string                      `json:"provider"`
	Connected         bool                        `json:"connected"`
	SetupStep         string                      `json:"setupStep,omitempty"`
	RequiredTotal     int                         `json:"requiredTotal"`
	RequiredConfirmed int                         `json:"requiredConfirmed"`
	CanCompleteSetup  bool                        `json:"canCompleteSetup"`
	Groups            []accountingMappingGroupRow `json:"groups"`
	Gaps              []accountingMappingRow      `json:"gaps"`
	HasMore           bool                        `json:"hasMore"`
	SetupPath         string                      `json:"setupPath"`
}

type accountingMappingDetail struct {
	Mapping accountingMappingRow  `json:"mapping"`
	Options []accountingRecordRow `json:"options,omitempty"`
}

func accountingSystemParam() map[string]any {
	return jsonschemautils.Enum(
		"The accounting system. Example: \"QuickBooksOnline\".",
		string(integration.TypeQuickBooksOnline),
	)
}

func targetTypeValues() []string {
	types := accountingsync.AllMappingTargetTypes()
	values := make([]string, 0, len(types))
	for _, targetType := range types {
		values = append(values, string(targetType))
	}
	return values
}

func requireAccountingSystem(params map[string]any) (integration.Type, error) {
	system := integration.Type(optionalString(params, paramAccountingSystem))
	if !accountingsync.SupportsAccountingSync(system) {
		return "", fmt.Errorf("system must be %q", integration.TypeQuickBooksOnline)
	}
	return system, nil
}

func optionalTargetType(params map[string]any) (accountingsync.MappingTargetType, error) {
	raw := optionalString(params, paramTargetType)
	if raw == "" {
		return "", nil
	}
	targetType := accountingsync.MappingTargetType(raw)
	if !targetType.IsValid() {
		return "", fmt.Errorf("targetType %q is not a mapping target type", raw)
	}
	return targetType, nil
}

func accountingMappingRowFrom(row *accountingsync.AccountingMapping) accountingMappingRow {
	out := accountingMappingRow{
		ID:          row.ID.String(),
		TargetType:  string(row.TargetType),
		Target:      row.TargetLabel,
		Key:         row.TrenovaKey,
		RecordKind:  string(row.ProviderKind),
		State:       string(row.State),
		Required:    row.IsRequired(),
		Source:      string(row.Source),
		Confidence:  row.Confidence,
		Reason:      row.Reason,
		ConfirmedOn: pointerDate(row.ConfirmedAt),
		CanCreate: row.ProviderKind.Creatable() &&
			row.State != accountingsync.MappingStateConfirmed,
	}
	if !row.TrenovaObjectID.IsNil() {
		out.RecordID = row.TrenovaObjectID.String()
	}
	if row.ExternalID != "" {
		out.MappedTo = &accountingRecordRow{ExternalID: row.ExternalID, Name: row.ExternalName}
	}
	if len(row.Signals.Candidates) > 0 {
		out.Candidates = make([]accountingCandidateRow, 0, len(row.Signals.Candidates))
		for _, candidate := range row.Signals.Candidates {
			out.Candidates = append(out.Candidates, accountingCandidateRow{
				ExternalID: candidate.ExternalID,
				Name:       candidate.Name,
				Score:      candidate.Score,
				Reason:     candidate.Reason,
			})
		}
	}
	return out
}

func accountingRecordRowFrom(ref *accountingsync.AccountingReferenceObject) accountingRecordRow {
	recordType := ref.AccountType
	if recordType == "" {
		recordType = ref.ItemType
	}
	return accountingRecordRow{
		ExternalID: ref.ExternalID,
		Name:       ref.Label(),
		Type:       recordType,
		Number:     ref.Number,
	}
}

type listAccountingMappingGapsTool struct {
	mappings accountingMappingReader
}

func newListAccountingMappingGapsTool(
	mappings accountingMappingReader,
) serviceports.AgentQueryTool {
	return &listAccountingMappingGapsTool{mappings: mappings}
}

func provideListAccountingMappingGapsTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentQueryTool {
	return newListAccountingMappingGapsTool(mappings)
}

func (t *listAccountingMappingGapsTool) Name() string { return "list_accounting_mapping_gaps" }

func (t *listAccountingMappingGapsTool) Description() string {
	return "List what still has to be matched to the accounting system before anything can sync. " +
		"It covers account roles, line types, accessorial charges, customers, carriers, terms " +
		"and payment methods that are unmatched or only proposed, and says how many required " +
		"mappings are confirmed and whether setup can finish. Each gap carries its id and the " +
		"candidates Trenova considered; pass them to get_accounting_mapping or " +
		"set_accounting_mapping."
}

func (t *listAccountingMappingGapsTool) SearchTerms() []string {
	return []string{"mapping", "unmapped", "setup", "match", "gaps"}
}

func (t *listAccountingMappingGapsTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemParam(),
		paramTargetType: jsonschemautils.Enum(
			"Only gaps of this kind. Leave it out for every kind.",
			targetTypeValues()...,
		),
		"requiredOnly": jsonschemautils.Boolean("Only the mappings setup cannot finish without."),
		paramLimit: jsonschemautils.Integer(
			"How many gaps to return, at most 50. Defaults to 25.",
		),
	}, paramAccountingSystem)
}

func (t *listAccountingMappingGapsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingIntegration})
}

func (t *listAccountingMappingGapsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, err
	}
	targetType, err := optionalTargetType(params.Params)
	if err != nil {
		return nil, err
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, mappingGapsDefaultLimit), 1),
		mappingGapsMaxLimit,
	)

	tenant := tenantOf(params)
	summary, err := t.mappings.Summary(ctx, tenant, system)
	if err != nil {
		return nil, err
	}

	result := &accountingMappingGapsResult{
		Provider:          summary.ProviderName,
		RequiredTotal:     summary.RequiredTotal,
		RequiredConfirmed: summary.RequiredConfirmed,
		CanCompleteSetup:  summary.CanCompleteSetup,
		Groups:            make([]accountingMappingGroupRow, 0, len(summary.Groups)),
		Gaps:              []accountingMappingRow{},
		SetupPath:         accountingsync.SetupPath(system),
	}
	for _, group := range summary.Groups {
		result.Groups = append(result.Groups, accountingMappingGroupRow{
			TargetType: string(group.TargetType),
			Unmatched:  group.Unmatched,
			Proposed:   group.Proposed,
			Confirmed:  group.Confirmed,
		})
	}
	if summary.Connection == nil {
		return result, nil
	}
	result.Connected = summary.Connection.IsActive()
	result.SetupStep = string(summary.Connection.SetupStep)

	req := &serviceports.ListAccountingMappingsRequest{
		TenantInfo:      tenant,
		IntegrationType: system,
		States: []accountingsync.MappingState{
			accountingsync.MappingStateUnmatched,
			accountingsync.MappingStateProposed,
		},
		RequiredOnly: optionalBool(params.Params, "requiredOnly"),
		Cursor:       pagination.CursorInfo{Limit: limit},
	}
	if targetType != "" {
		req.TargetTypes = []accountingsync.MappingTargetType{targetType}
	}
	page, err := t.mappings.ListMappings(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, row := range page.Items {
		result.Gaps = append(result.Gaps, accountingMappingRowFrom(row))
	}
	result.HasMore = page.HasNextPage

	return result, nil
}

type getAccountingMappingTool struct {
	mappings accountingMappingReader
}

func newGetAccountingMappingTool(mappings accountingMappingReader) serviceports.AgentQueryTool {
	return &getAccountingMappingTool{mappings: mappings}
}

func provideGetAccountingMappingTool(
	mappings serviceports.AccountingMappingService,
) serviceports.AgentQueryTool {
	return newGetAccountingMappingTool(mappings)
}

func (t *getAccountingMappingTool) Name() string { return "get_accounting_mapping" }

func (t *getAccountingMappingTool) Description() string {
	return "Get which accounting system record one Trenova record or setting is sent as. " +
		"Name it by mappingId, or by targetType with recordId for a customer, carrier or " +
		"accessorial charge, or with key for an account role, line type, term or payment " +
		"method. With search, it also returns up to ten usable accounting records of the " +
		"right kind whose names match, so you can choose one for set_accounting_mapping."
}

func (t *getAccountingMappingTool) SearchTerms() []string {
	return []string{"account", "post", "posts", "revenue", "mapped", "item", "vendor"}
}

func (t *getAccountingMappingTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemParam(),
		paramMappingID: jsonschemautils.Text(
			"The mapping's id from list_accounting_mapping_gaps.",
		),
		paramTargetType: jsonschemautils.Enum(
			"The kind of Trenova record or setting, when naming it by recordId or key.",
			targetTypeValues()...,
		),
		paramRecordID: jsonschemautils.Text(
			"The Trenova customer, carrier or accessorial charge id, from " +
				"list_customers, list_carriers or list_accessorial_charges.",
		),
		paramMappingKey: jsonschemautils.Text(
			"The setting's key from list_accounting_mapping_gaps, for the " +
				"other target types. Examples: " +
				"\"ARAccount\", \"RevenueAccount\", \"Freight\", \"Net30\", \"ACH\".",
		),
		"search": jsonschemautils.Text("Words to find in the accounting system's record names."),
	}, paramAccountingSystem)
}

func (t *getAccountingMappingTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingIntegration})
}

func (t *getAccountingMappingTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, err
	}
	find, err := mappingLookup(params, system)
	if err != nil {
		return nil, err
	}

	row, err := t.mappings.FindMapping(ctx, find)
	if err != nil {
		return nil, err
	}
	detail := &accountingMappingDetail{Mapping: accountingMappingRowFrom(row)}

	search := optionalString(params.Params, "search")
	if search == "" {
		return detail, nil
	}
	refs, err := t.mappings.SearchReference(ctx, &serviceports.SearchAccountingReferenceRequest{
		TenantInfo:      find.TenantInfo,
		IntegrationType: system,
		Kind:            row.ProviderKind,
		Query:           search,
		UsableOnly:      true,
		Limit:           mappingOptionsLimit,
	})
	if err != nil {
		return nil, err
	}
	detail.Options = make([]accountingRecordRow, 0, len(refs))
	for _, ref := range refs {
		detail.Options = append(detail.Options, accountingRecordRowFrom(ref))
	}
	return detail, nil
}

func mappingLookup(
	params *serviceports.QueryToolParams,
	system integration.Type,
) (*serviceports.SetAccountingMappingRequest, error) {
	mappingID, err := optionalID(params.Params, paramMappingID)
	if err != nil {
		return nil, err
	}
	recordID, err := optionalID(params.Params, paramRecordID)
	if err != nil {
		return nil, err
	}
	req := &serviceports.SetAccountingMappingRequest{
		TenantInfo:      tenantOf(params),
		IntegrationType: system,
		MappingID:       mappingID,
		TargetType: accountingsync.MappingTargetType(
			optionalString(params.Params, paramTargetType),
		),
		TrenovaObjectID: recordID,
		TrenovaKey:      optionalString(params.Params, paramMappingKey),
	}
	if err = req.ValidateTarget(); err != nil {
		return nil, err
	}
	return req, nil
}
