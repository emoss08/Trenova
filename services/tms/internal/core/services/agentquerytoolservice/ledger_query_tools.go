package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	paramManualJournalID    = "manualJournalId"
	fieldJournalDescription = "description"
)

var manualJournalStatuses = []string{
	string(manualjournal.StatusDraft),
	string(manualjournal.StatusPendingApproval),
	string(manualjournal.StatusApproved),
	string(manualjournal.StatusRejected),
	string(manualjournal.StatusCancelled),
	string(manualjournal.StatusPosted),
}

var journalReversalStatuses = []string{
	string(journalreversal.StatusRequested),
	string(journalreversal.StatusPendingApproval),
	string(journalreversal.StatusApproved),
	string(journalreversal.StatusRejected),
	string(journalreversal.StatusCancelled),
	string(journalreversal.StatusPosted),
}

func ledgerToolProviders() []any {
	return []any{
		newListManualJournalsTool,
		newGetManualJournalTool,
		newListJournalReversalsTool,
	}
}

type manualJournalRow struct {
	ID              string       `json:"id"`
	RequestNumber   string       `json:"requestNumber"`
	Status          string       `json:"status"`
	Description     string       `json:"description"`
	AccountingDate  optionalDate `json:"accountingDate"`
	FiscalPeriodID  string       `json:"fiscalPeriodId"`
	CurrencyCode    string       `json:"currencyCode"`
	TotalDebit      string       `json:"totalDebit,omitempty"`
	TotalCredit     string       `json:"totalCredit,omitempty"`
	Balanced        bool         `json:"balanced"`
	ApprovedAt      optionalDate `json:"approvedAt"`
	PostedBatchID   string       `json:"postedBatchId,omitempty"`
	Reason          string       `json:"reason,omitempty"`
	RejectionReason string       `json:"rejectionReason,omitempty"`
	CancelReason    string       `json:"cancelReason,omitempty"`
}

func manualJournalRowFrom(entity *manualjournal.Request, gate *fieldGate) manualJournalRow {
	row := manualJournalRow{
		ID:             entity.ID.String(),
		RequestNumber:  entity.RequestNumber,
		Status:         string(entity.Status),
		Description:    entity.Description,
		AccountingDate: recordedDate(entity.AccountingDate),
		FiscalPeriodID: pulidString(entity.RequestedFiscalPeriodID),
		CurrencyCode:   entity.CurrencyCode,
		Balanced:       entity.IsBalanced(),
		ApprovedAt:     expectedDate(typeutils.ValueOrZero(entity.ApprovedAt), absentNotApproved),
		PostedBatchID:  pulidString(entity.PostedBatchID),
	}
	if gate.show("totalDebit", "totalDebit") {
		row.TotalDebit = minorText(entity.TotalDebit)
	}
	if gate.show("totalCredit", "totalCredit") {
		row.TotalCredit = minorText(entity.TotalCredit)
	}
	if entity.Reason != "" && gate.show("reason", "reason") {
		row.Reason = entity.Reason
	}
	if entity.RejectionReason != "" && gate.show("rejectionReason", "rejectionReason") {
		row.RejectionReason = entity.RejectionReason
	}
	if entity.CancelReason != "" && gate.show("cancelReason", "cancelReason") {
		row.CancelReason = entity.CancelReason
	}

	return row
}

func newListManualJournalsTool(
	repo repositories.ManualJournalRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_manual_journals",
		entityPlural: "manual journals",
		summary: "List manual journal requests, newest first, with their number, status, " +
			"accounting day and totals, and whether they balance. Narrow by status to find " +
			"drafts to finish, journals awaiting approval, or approved ones still to post. " +
			"get_manual_journal opens one with its lines.",
		resource: permission.ResourceManualJournal,
		config:   querybuilder.GetFieldConfiguration((*manualjournal.Request)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: manualJournalStatuses,
				Note:   "Approved journals are waiting to be posted",
			},
			{Name: "requestNumber", Kind: filterText, Sortable: true},
			{Name: fieldJournalDescription, Kind: filterText},
			{Name: "accountingDate", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListManualJournalRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *manualjournal.Request) any {
				return manualJournalRowFrom(item, gate)
			}), nil
		},
	})
}

type manualJournalLineRow struct {
	LineNumber  int    `json:"lineNumber"`
	AccountID   string `json:"glAccountId"`
	AccountCode string `json:"accountCode,omitempty"`
	AccountName string `json:"accountName,omitempty"`
	Description string `json:"description"`
	CustomerID  string `json:"customerId,omitempty"`
	LocationID  string `json:"locationId,omitempty"`
	Debit       string `json:"debit,omitempty"`
	Credit      string `json:"credit,omitempty"`
}

type manualJournalView struct {
	manualJournalRow

	RejectedAt  optionalDate           `json:"rejectedAt"`
	CancelledAt optionalDate           `json:"cancelledAt"`
	Lines       []manualJournalLineRow `json:"lines"`
	Withheld    []string               `json:"withheldByAccess,omitempty"`
}

type manualJournalReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetManualJournalByIDRequest,
	) (*manualjournal.Request, error)
}

type getManualJournalTool struct {
	journals manualJournalReader
	access   fieldAccess
}

func newGetManualJournalTool(
	repo repositories.ManualJournalRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getManualJournalTool{journals: repo, access: newFieldAccess(permissions)}
}

func (t *getManualJournalTool) Name() string { return "get_manual_journal" }

func (t *getManualJournalTool) Description() string {
	return "Retrieve one manual journal request by id with its lines, the account each " +
		"posts to and its debit or credit. It also says who approved, rejected or cancelled " +
		"it and why. Use list_manual_journals first when you have a number."
}

func (t *getManualJournalTool) ParamSchema() map[string]any {
	return idSchema(paramManualJournalID, agenttoolschema.RecordIDText(
		permission.ResourceManualJournal,
		"The manual journal's id, from list_manual_journals or "+onThePage,
	))
}

func (t *getManualJournalTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceManualJournal})
}

func (t *getManualJournalTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}

	entity, err := t.journals.GetByID(ctx, repositories.GetManualJournalByIDRequest{
		ID:         id,
		TenantInfo: tenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceManualJournal)
	view := manualJournalView{
		manualJournalRow: manualJournalRowFrom(entity, gate),
		RejectedAt:       expectedDate(typeutils.ValueOrZero(entity.RejectedAt), "not rejected"),
		CancelledAt:      expectedDate(typeutils.ValueOrZero(entity.CancelledAt), "not cancelled"),
		Lines:            make([]manualJournalLineRow, 0, min(len(entity.Lines), maxJournalLines)),
	}

	showDebit := gate.show("debitAmount", "lines.debit")
	showCredit := gate.show("creditAmount", "lines.credit")
	for _, line := range entity.Lines {
		if line == nil || len(view.Lines) == maxJournalLines {
			continue
		}
		row := manualJournalLineRow{
			LineNumber:  line.LineNumber,
			AccountID:   line.GLAccountID.String(),
			Description: line.Description,
			CustomerID:  pulidString(line.CustomerID),
			LocationID:  pulidString(line.LocationID),
		}
		if line.GLAccount != nil {
			row.AccountCode = line.GLAccount.AccountCode
			row.AccountName = line.GLAccount.Name
		}
		if showDebit && line.DebitAmount != 0 {
			row.Debit = minorText(line.DebitAmount)
		}
		if showCredit && line.CreditAmount != 0 {
			row.Credit = minorText(line.CreditAmount)
		}
		view.Lines = append(view.Lines, row)
	}
	view.Withheld = gate.Withheld()

	return view, nil
}

type journalReversalRow struct {
	ID                     string       `json:"id"`
	Status                 string       `json:"status"`
	OriginalJournalEntryID string       `json:"originalJournalEntryId"`
	ReversalJournalEntryID string       `json:"reversalJournalEntryId,omitempty"`
	AccountingDate         optionalDate `json:"accountingDate"`
	FiscalPeriodID         string       `json:"fiscalPeriodId"`
	ReasonCode             string       `json:"reasonCode"`
	ReasonText             string       `json:"reasonText,omitempty"`
	ApprovedAt             optionalDate `json:"approvedAt"`
	PostedAt               optionalDate `json:"postedAt"`
	RejectionReason        string       `json:"rejectionReason,omitempty"`
	CancelReason           string       `json:"cancelReason,omitempty"`
}

func newListJournalReversalsTool(
	repo repositories.JournalReversalRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_journal_reversals",
		entityPlural: "journal reversals",
		summary: "List requests to reverse posted journal entries, newest first, with the " +
			"entry each reverses, its status, the day it books and why. Narrow by status to " +
			"find reversals awaiting approval or approved ones still to post.",
		resource: permission.ResourceJournalReversal,
		config:   querybuilder.GetFieldConfiguration((*journalreversal.Reversal)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: journalReversalStatuses,
				Note:   "Approved reversals are waiting to be posted",
			},
			{Name: "reasonCode", Kind: filterText},
			{Name: "originalJournalEntryId", Kind: filterText},
			{Name: "requestedAccountingDate", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListJournalReversalsRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *journalreversal.Reversal) any {
				row := journalReversalRow{
					ID:                     item.ID.String(),
					Status:                 string(item.Status),
					OriginalJournalEntryID: item.OriginalJournalEntryID.String(),
					ReversalJournalEntryID: pulidString(item.ReversalJournalEntryID),
					AccountingDate:         recordedDate(item.RequestedAccountingDate),
					FiscalPeriodID:         pulidString(item.ResolvedFiscalPeriodID),
					ReasonCode:             item.ReasonCode,
					ApprovedAt: expectedDate(
						typeutils.ValueOrZero(item.ApprovedAt),
						absentNotApproved,
					),
					PostedAt: expectedDate(typeutils.ValueOrZero(item.PostedAt), absentNotPosted),
				}
				if gate.show("reasonText", "reasonText") {
					row.ReasonText = item.ReasonText
				}
				if item.RejectionReason != "" &&
					gate.show("rejectionReason", "rejectionReason") {
					row.RejectionReason = item.RejectionReason
				}
				if item.CancelReason != "" && gate.show("cancelReason", "cancelReason") {
					row.CancelReason = item.CancelReason
				}

				return row
			}), nil
		},
	})
}
