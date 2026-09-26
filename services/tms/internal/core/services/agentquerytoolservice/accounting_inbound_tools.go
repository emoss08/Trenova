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
	paramInboundStatus = "status"
	paramInboundKind   = "kind"
	paramInboundReason = "reason"
	paramInboundSearch = "search"

	inboundDefaultLimit = 25
	inboundMaxLimit     = 50

	accountingInboundPath = "/accounting/sync/inbound"
)

type accountingInboundReader interface {
	List(
		ctx context.Context,
		req *serviceports.ListAccountingInboundChangesRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error)
}

type accountingInboundLineRow struct {
	DocumentKind   string `json:"documentKind"`
	DocumentID     string `json:"documentId,omitempty"`
	DocumentNumber string `json:"documentNumber,omitempty"`
	Amount         string `json:"amount"`
	OpenInTrenova  string `json:"openInTrenova,omitempty"`
}

type accountingInboundRow struct {
	ID              string                     `json:"id"`
	Kind            string                     `json:"kind"`
	Status          string                     `json:"status"`
	Reason          string                     `json:"reason,omitempty"`
	Resolution      string                     `json:"resolution,omitempty"`
	ProviderNumber  string                     `json:"providerNumber,omitempty"`
	ProviderURL     string                     `json:"providerUrl,omitempty"`
	Party           string                     `json:"party,omitempty"`
	PartyID         string                     `json:"partyId,omitempty"`
	PaidOn          optionalDate               `json:"paidOn"`
	Amount          string                     `json:"amount"`
	ReferenceNumber string                     `json:"referenceNumber,omitempty"`
	Method          string                     `json:"method,omitempty"`
	ChangedBy       string                     `json:"changedBy,omitempty"`
	Pays            []accountingInboundLineRow `json:"pays"`
	DetectedOn      optionalDate               `json:"detectedOn"`
	CanApply        bool                       `json:"canApply"`
	CanIgnore       bool                       `json:"canIgnore"`
}

type accountingInboundResult struct {
	Changes    []accountingInboundRow `json:"changes"`
	HasMore    bool                   `json:"hasMore"`
	NextCursor string                 `json:"nextCursor,omitempty"`
	PagePath   string                 `json:"pagePath"`
}

func accountingInboundRowFrom(change *accountingsync.AccountingInboundChange) accountingInboundRow {
	row := accountingInboundRow{
		ID:              change.ID.String(),
		Kind:            string(change.Kind),
		Status:          string(change.Status),
		Reason:          string(change.Reason),
		Resolution:      change.Resolution,
		ProviderNumber:  change.ExternalNumber,
		ProviderURL:     change.ExternalURL,
		Party:           change.PartyName,
		PaidOn:          recordedDate(change.TxnDate),
		Amount:          money.FormatMinor(change.AmountMinor, change.CurrencyCode),
		ReferenceNumber: change.Document.ReferenceNumber,
		Method:          change.Document.MethodName,
		ChangedBy:       change.ProviderModifiedBy,
		Pays:            make([]accountingInboundLineRow, 0, len(change.Document.Lines)),
		DetectedOn:      recordedDate(change.DetectedAt),
		CanApply:        change.CanApply() == nil,
		CanIgnore:       change.Status.IsOpen(),
	}
	if !change.PartyObjectID.IsNil() {
		row.PartyID = change.PartyObjectID.String()
	}
	for _, line := range change.Document.Lines {
		lineRow := accountingInboundLineRow{
			DocumentKind:   string(line.DocumentKind),
			DocumentNumber: line.ObjectNumber,
			Amount:         money.FormatMinor(line.AmountMinor, change.CurrencyCode),
		}
		if line.Matched() {
			lineRow.DocumentID = line.ObjectID.String()
			lineRow.OpenInTrenova = money.FormatMinor(line.OpenMinor, change.CurrencyCode)
		}
		row.Pays = append(row.Pays, lineRow)
	}
	return row
}

type listAccountingInboundChangesTool struct {
	inbound accountingInboundReader
}

func newListAccountingInboundChangesTool(
	inbound accountingInboundReader,
) serviceports.AgentQueryTool {
	return &listAccountingInboundChangesTool{inbound: inbound}
}

func provideListAccountingInboundChangesTool(
	inbound serviceports.AccountingInboundService,
) serviceports.AgentQueryTool {
	return newListAccountingInboundChangesTool(inbound)
}

func (t *listAccountingInboundChangesTool) Name() string {
	return "list_accounting_inbound_changes"
}

func (t *listAccountingInboundChangesTool) Description() string {
	return "List payments recorded in the accounting system against invoices or bills " +
		"Trenova sent, newest first. Each row says what it pays, whether it was applied in " +
		"Trenova, and if not why, in plain language. Filter by status, such as Proposed for " +
		"what waits on a person, by kind, or by reason. Pass a Proposed row's id to " +
		"apply_accounting_inbound_change or ignore_accounting_inbound_change."
}

func (t *listAccountingInboundChangesTool) SearchTerms() []string {
	return []string{"paid in quickbooks", "payment from the books", "inbound", "bill payment"}
}

func (t *listAccountingInboundChangesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramAccountingSystem: accountingSystemParam(),
		paramInboundStatus: jsonschemautils.DescribedArray(
			"Only payments in these statuses. Proposed ones wait for a person.",
			jsonschemautils.Enum(
				"A status.",
				sliceutils.Strings(accountingsync.AllInboundChangeStatuses())...,
			),
			len(accountingsync.AllInboundChangeStatuses()),
		),
		paramInboundKind: jsonschemautils.DescribedArray(
			"Only customer payments or only bill payments.",
			jsonschemautils.Enum(
				"A kind.",
				sliceutils.Strings(accountingsync.AllInboundChangeKinds())...,
			),
			len(accountingsync.AllInboundChangeKinds()),
		),
		paramInboundReason: jsonschemautils.DescribedArray(
			"Only payments held back for these reasons.",
			jsonschemautils.Enum(
				"A reason.",
				sliceutils.Strings(accountingsync.AllInboundChangeReasons())...,
			),
			len(accountingsync.AllInboundChangeReasons()),
		),
		paramInboundSearch: jsonschemautils.Text(
			"Words to find in the payment's number or the customer or vendor name.",
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many payments to return, at most %d. Defaults to %d.",
			inboundMaxLimit, inboundDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	}, paramAccountingSystem)
}

func (t *listAccountingInboundChangesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountingSync})
}

func (t *listAccountingInboundChangesTool) request(
	params *serviceports.QueryToolParams,
) (*serviceports.ListAccountingInboundChangesRequest, error) {
	system, err := requireAccountingSystem(params.Params)
	if err != nil {
		return nil, err
	}
	statuses, err := optionalEnums(
		params.Params,
		paramInboundStatus,
		accountingsync.InboundChangeStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	kinds, err := optionalEnums(
		params.Params,
		paramInboundKind,
		accountingsync.InboundChangeKind.IsValid,
	)
	if err != nil {
		return nil, err
	}
	reasons, err := optionalEnums(
		params.Params,
		paramInboundReason,
		accountingsync.InboundChangeReason.IsValid,
	)
	if err != nil {
		return nil, err
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, inboundDefaultLimit), 1),
		inboundMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}
	cursor.IncludeTotalCount = false

	return &serviceports.ListAccountingInboundChangesRequest{
		TenantInfo:      tenantOf(params),
		IntegrationType: system,
		Statuses:        statuses,
		Kinds:           kinds,
		Reasons:         reasons,
		Search:          optionalString(params.Params, paramInboundSearch),
		Cursor:          cursor,
	}, nil
}

func (t *listAccountingInboundChangesTool) Query(
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

	page, err := t.inbound.List(ctx, req)
	if err != nil {
		return nil, err
	}

	result := &accountingInboundResult{
		Changes:  make([]accountingInboundRow, 0, len(page.Items)),
		HasMore:  page.HasNextPage,
		PagePath: accountingInboundPath,
	}
	for _, change := range page.Items {
		if change == nil {
			continue
		}
		result.Changes = append(result.Changes, accountingInboundRowFrom(change))
	}
	if result.NextCursor, err = page.NextCursor(); err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return result, nil
}
