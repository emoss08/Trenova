package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramDriftStatus     = "status"
	paramDriftKind       = "kind"
	paramDriftObjectType = "objectType"
	paramDriftSearch     = "search"

	driftDefaultLimit = 25
	driftMaxLimit     = 50

	accountingDriftPath = "/accounting/sync/drift"
)

type accountingDriftReader interface {
	List(
		ctx context.Context,
		req *serviceports.ListAccountingDriftFindingsRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingDriftFinding], error)
	Overview(
		ctx context.Context,
		req *serviceports.AccountingDriftOverviewRequest,
	) (*serviceports.AccountingDriftOverview, error)
}

type accountingDriftLineRow struct {
	DocumentType   string `json:"documentType"`
	DocumentID     string `json:"documentId"`
	DocumentNumber string `json:"documentNumber"`
	InTrenova      string `json:"inTrenova"`
	InProvider     string `json:"inProvider"`
}

type accountingDriftRow struct {
	ID              string                   `json:"id"`
	Kind            string                   `json:"kind"`
	Status          string                   `json:"status"`
	Resolution      string                   `json:"resolution,omitempty"`
	Note            string                   `json:"note,omitempty"`
	ObjectType      string                   `json:"objectType"`
	ObjectID        string                   `json:"objectId"`
	Number          string                   `json:"number,omitempty"`
	Party           string                   `json:"party,omitempty"`
	ProviderURL     string                   `json:"providerUrl,omitempty"`
	InTrenova       string                   `json:"inTrenova,omitempty"`
	InProvider      string                   `json:"inProvider,omitempty"`
	Difference      string                   `json:"difference,omitempty"`
	WithinTolerance bool                     `json:"withinTolerance"`
	TrenovaState    string                   `json:"trenovaState,omitempty"`
	ProviderState   string                   `json:"providerState,omitempty"`
	ChangedBy       string                   `json:"changedBy,omitempty"`
	ChangedOn       optionalDate             `json:"changedOn"`
	DetectedOn      optionalDate             `json:"detectedOn"`
	Documents       []accountingDriftLineRow `json:"documents,omitempty"`
	Fixes           []string                 `json:"fixes"`
	WaitingOnSend   bool                     `json:"waitingOnSend"`
}

type accountingDriftResult struct {
	Findings   []accountingDriftRow `json:"findings"`
	Tolerance  string               `json:"tolerance"`
	CheckedOn  optionalDate         `json:"checkedOn"`
	CheckError string               `json:"checkError,omitempty"`
	HasMore    bool                 `json:"hasMore"`
	NextCursor string               `json:"nextCursor,omitempty"`
	PagePath   string               `json:"pagePath"`
}

func formatDriftMinor(minor *int64, currency string) string {
	if minor == nil {
		return ""
	}
	return money.FormatMinor(*minor, currency)
}

func accountingDriftRowFrom(
	finding *accountingsync.AccountingDriftFinding,
	toleranceMinor int64,
) accountingDriftRow {
	currency := finding.CurrencyCode
	row := accountingDriftRow{
		ID:              finding.ID.String(),
		Kind:            string(finding.Kind),
		Status:          string(finding.Status),
		Resolution:      string(finding.Resolution),
		Note:            finding.ResolutionNote,
		ObjectType:      string(finding.ObjectType),
		ObjectID:        finding.ObjectID.String(),
		Number:          finding.ObjectNumber,
		Party:           finding.PartyName,
		ProviderURL:     finding.ExternalURL,
		InTrenova:       formatDriftMinor(finding.TrenovaMinor, currency),
		InProvider:      formatDriftMinor(finding.ProviderMinor, currency),
		Difference:      formatDriftMinor(finding.DifferenceMinor, currency),
		WithinTolerance: finding.WithinTolerance(toleranceMinor),
		TrenovaState:    finding.TrenovaState,
		ProviderState:   finding.ProviderState,
		ChangedBy:       finding.ProviderModifiedBy,
		DetectedOn:      recordedDate(finding.DetectedAt),
		Fixes:           []string{},
		WaitingOnSend:   finding.Pushed(),
	}
	if finding.ProviderModifiedAt != nil {
		row.ChangedOn = recordedDate(*finding.ProviderModifiedAt)
	}
	if finding.IsOpen() {
		row.Fixes = sliceutils.Strings(finding.Directions())
	}
	for _, line := range finding.Detail {
		row.Documents = append(row.Documents, accountingDriftLineRow{
			DocumentType:   string(line.ObjectType),
			DocumentID:     line.ObjectID.String(),
			DocumentNumber: line.ObjectNumber,
			InTrenova:      money.FormatMinor(line.TrenovaMinor, currency),
			InProvider:     money.FormatMinor(line.ProviderMinor, currency),
		})
	}
	return row
}

type listAccountingDriftFindingsTool struct {
	drift accountingDriftReader
}

func newListAccountingDriftFindingsTool(drift accountingDriftReader) serviceports.AgentQueryTool {
	return &listAccountingDriftFindingsTool{drift: drift}
}

func provideListAccountingDriftFindingsTool(
	drift serviceports.AccountingDriftService,
) serviceports.AgentQueryTool {
	return newListAccountingDriftFindingsTool(drift)
}

func (t *listAccountingDriftFindingsTool) Name() string {
	return "list_accounting_drift_findings"
}

func (t *listAccountingDriftFindingsTool) Description() string {
	return "List drift: documents Trenova sent to the accounting system that differ there " +
		"now, newest first. Each row shows both values, who changed it there and when, " +
		"whether the difference is within the reconciliation tolerance, and the fixes it " +
		"offers. Filter by status, such as Open, by kind, or by document type. Pass an Open " +
		"row's id to resolve_accounting_drift or dismiss_accounting_drift."
}

func (t *listAccountingDriftFindingsTool) SearchTerms() []string {
	return []string{
		"drift",
		"changed after it was sent",
		"edited in the accounting system",
		"reconciliation differences",
	}
}

func (t *listAccountingDriftFindingsTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemParam(),
		paramDriftStatus: jsonschemautils.DescribedArray(
			"Only findings in these statuses. Open ones wait for a fix or a dismissal.",
			jsonschemautils.Enum(
				"A status.",
				sliceutils.Strings(accountingsync.AllDriftStatuses())...,
			),
			len(accountingsync.AllDriftStatuses()),
		),
		paramDriftKind: jsonschemautils.DescribedArray(
			"Only these kinds of difference.",
			jsonschemautils.Enum("A kind.", sliceutils.Strings(accountingsync.AllDriftKinds())...),
			len(accountingsync.AllDriftKinds()),
		),
		paramDriftObjectType: jsonschemautils.DescribedArray(
			"Only these document types. Customer means a customer's balance.",
			jsonschemautils.Enum(
				"A document type.",
				sliceutils.Strings(append(
					[]accountingsync.SyncObjectType{accountingsync.SyncObjectCustomer},
					accountingsync.DriftObjectTypes()...,
				))...,
			),
			len(accountingsync.DriftObjectTypes())+1,
		),
		paramDriftSearch: jsonschemautils.Text(
			"Words to find in the document number, the party name or who changed it.",
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many findings to return, at most %d. Defaults to %d.",
			driftMaxLimit, driftDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	}, paramAccountingSystem)
}

func (t *listAccountingDriftFindingsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *listAccountingDriftFindingsTool) request(
	params *serviceports.QueryToolParams,
) (*serviceports.ListAccountingDriftFindingsRequest, error) {
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, err
	}
	statuses, err := optionalEnums(
		params.Params,
		paramDriftStatus,
		accountingsync.DriftStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	kinds, err := optionalEnums(params.Params, paramDriftKind, accountingsync.DriftKind.IsValid)
	if err != nil {
		return nil, err
	}
	objectTypes, err := optionalEnums(
		params.Params,
		paramDriftObjectType,
		accountingsync.SyncObjectType.IsValid,
	)
	if err != nil {
		return nil, err
	}
	limit := min(max(optionalInt(params.Params, paramLimit, driftDefaultLimit), 1), driftMaxLimit)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}
	cursor.IncludeTotalCount = false

	return &serviceports.ListAccountingDriftFindingsRequest{
		TenantInfo:      tenantOf(params),
		IntegrationType: system,
		Statuses:        statuses,
		Kinds:           kinds,
		ObjectTypes:     objectTypes,
		Search:          optionalString(params.Params, paramDriftSearch),
		Cursor:          cursor,
	}, nil
}

func (t *listAccountingDriftFindingsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}
	overview, err := t.drift.Overview(ctx, &serviceports.AccountingDriftOverviewRequest{
		TenantInfo:      req.TenantInfo,
		IntegrationType: req.IntegrationType,
	})
	if err != nil {
		return nil, err
	}
	page, err := t.drift.List(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &accountingDriftResult{
		Findings:   make([]accountingDriftRow, 0, len(page.Items)),
		Tolerance:  money.FormatMinor(overview.ToleranceMinor, overview.CurrencyCode),
		CheckError: overview.CheckError,
		HasMore:    page.HasNextPage,
		PagePath:   accountingDriftPath,
	}
	if overview.CheckedAt != nil {
		result.CheckedOn = recordedDate(*overview.CheckedAt)
	}
	for _, finding := range page.Items {
		if finding == nil {
			continue
		}
		result.Findings = append(
			result.Findings,
			accountingDriftRowFrom(finding, overview.ToleranceMinor),
		)
	}
	if result.NextCursor, err = page.NextCursor(); err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return result, nil
}
