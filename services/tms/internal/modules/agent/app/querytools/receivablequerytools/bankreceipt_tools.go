package receivablequerytools

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxBankReceiptRows     = 50
	defaultBankReceiptRows = 20
)

// bankReceiptReader is the slice of the bank receipt service the read tools
// use: one receipt, the exceptions, and the scored candidates for one.
type bankReceiptReader interface {
	Get(
		ctx context.Context,
		req *bankreceiptports.GetBankReceiptRequest,
	) (*bankreceipt.BankReceipt, error)
	ListExceptions(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) ([]*bankreceipt.BankReceipt, error)
	SuggestMatches(
		ctx context.Context,
		req *bankreceiptports.GetBankReceiptRequest,
	) ([]*bankreceiptports.BankReceiptMatchSuggestion, error)
}

// workItemReader reads the reconciliation queue entry behind a receipt.
type workItemReader interface {
	ListActive(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) ([]*bankreceiptworkitem.WorkItem, error)
	GetActiveByReceiptID(
		ctx context.Context,
		tenant pagination.TenantInfo,
		receiptID pulid.ID,
	) (*bankreceiptworkitem.WorkItem, error)
}

// bankReceiptRow is a receipt in the words the model needs: when the money
// arrived, how much, what the bank said about it, and why it is waiting.
type bankReceiptRow struct {
	ID                       string                 `json:"id"`
	ReceiptDate              querytool.OptionalDate `json:"receiptDate"`
	Amount                   string                 `json:"amount"`
	ReferenceNumber          string                 `json:"referenceNumber,omitempty"`
	Memo                     string                 `json:"memo,omitempty"`
	Status                   string                 `json:"status"`
	ExceptionReason          string                 `json:"exceptionReason,omitempty"`
	MatchedCustomerPaymentID string                 `json:"matchedCustomerPaymentId,omitempty"`
	MatchedOn                querytool.OptionalDate `json:"matchedOn"`
	ImportBatchID            string                 `json:"importBatchId,omitempty"`
	WorkItem                 *workItemRow           `json:"workItem,omitempty"`
}

type workItemRow struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	AssignedToUserID string `json:"assignedToUserId,omitempty"`
	ResolutionType   string `json:"resolutionType,omitempty"`
	ResolutionNote   string `json:"resolutionNote,omitempty"`
}

func bankReceiptRowFrom(
	entity *bankreceipt.BankReceipt,
	item *bankreceiptworkitem.WorkItem,
) bankReceiptRow {
	row := bankReceiptRow{
		ID:              entity.ID.String(),
		ReceiptDate:     querytool.RecordedDate(entity.ReceiptDate),
		Amount:          money.DecimalFromMinor(entity.AmountMinor).StringFixed(2),
		ReferenceNumber: entity.ReferenceNumber,
		Memo:            entity.Memo,
		Status:          string(entity.Status),
		ExceptionReason: entity.ExceptionReason,
		MatchedOn: querytool.ExpectedDate(
			querytool.PointerSeconds(entity.MatchedAt),
			"not matched",
		),
	}
	if entity.MatchedCustomerPaymentID.IsNotNil() {
		row.MatchedCustomerPaymentID = entity.MatchedCustomerPaymentID.String()
	}
	if entity.ImportBatchID.IsNotNil() {
		row.ImportBatchID = entity.ImportBatchID.String()
	}
	if item != nil {
		row.WorkItem = &workItemRow{
			ID:             item.ID.String(),
			Status:         string(item.Status),
			ResolutionType: string(item.ResolutionType),
			ResolutionNote: item.ResolutionNote,
		}
		if item.AssignedToUserID.IsNotNil() {
			row.WorkItem.AssignedToUserID = item.AssignedToUserID.String()
		}
	}

	return row
}

var errNoActiveWorkItem = errors.New("the receipt has no open queue entry")

// activeWorkItem reads the queue entry for a receipt, and reads "none" as
// errNoActiveWorkItem rather than as a failure: a receipt with no open item is
// the normal state of a matched one.
func activeWorkItem(
	ctx context.Context,
	items workItemReader,
	tenant pagination.TenantInfo,
	receiptID pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	if items == nil {
		return nil, errNoActiveWorkItem
	}

	item, err := items.GetActiveByReceiptID(ctx, tenant, receiptID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errNoActiveWorkItem
		}

		return nil, err
	}

	return item, nil
}

type listBankReceiptExceptionsTool struct {
	receipts bankReceiptReader
	items    workItemReader
}

func NewListBankReceiptExceptionsTool(
	receipts bankReceiptReader,
	items workItemReader,
) agenttoolports.AgentQueryTool {
	return &listBankReceiptExceptionsTool{receipts: receipts, items: items}
}

func (t *listBankReceiptExceptionsTool) Name() string {
	return "receivable_list_bank_receipt_exceptions"
}

func (t *listBankReceiptExceptionsTool) Description() string {
	return "List the bank receipts that could not be matched to a customer payment on " +
		"their own and are waiting in the reconciliation queue. Each has its amount, the " +
		"bank's reference and memo, why it was not matched, and the queue entry with who " +
		"it is assigned to. Narrow with query to match the reference, memo or reason. " +
		"Use receivable_get_bank_receipt for one receipt's candidate payments."
}

func (t *listBankReceiptExceptionsTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"query": map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "Words to look for in the reference, the memo or the exception reason.",
			},
			querytool.ParamLimit: querytool.LimitSchema(defaultBankReceiptRows, maxBankReceiptRows),
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listBankReceiptExceptionsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceBankReceipt,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceBankReceipt,
		Rationale: "Lists bank receipts whose memos the payers wrote; nothing changes and " +
			"nothing is sent.",
	})
}

func (t *listBankReceiptExceptionsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	limit := querytool.OptionalInt(params.Params, querytool.ParamLimit, defaultBankReceiptRows)
	if limit <= 0 || limit > maxBankReceiptRows {
		limit = maxBankReceiptRows
	}
	query := strings.ToLower(strings.TrimSpace(querytool.OptionalString(params.Params, "query")))

	criteria := filtercatalog.NewCriteria("unmatched bank receipts").At(querytool.ClockFor(params))
	criteria.Text(query)

	tenant := querytool.TenantOf(params)
	receipts, err := t.receipts.ListExceptions(ctx, tenant)
	if err != nil {
		return nil, err
	}

	itemsByReceipt := make(map[pulid.ID]*bankreceiptworkitem.WorkItem)
	if t.items != nil {
		active, listErr := t.items.ListActive(ctx, tenant)
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range active {
			if item != nil {
				itemsByReceipt[item.BankReceiptID] = item
			}
		}
	}

	rows := make([]bankReceiptRow, 0, len(receipts))
	shown := make([]agentrecord.RecordRef, 0, limit)
	matched := 0
	for _, receipt := range receipts {
		if receipt == nil || !receiptMatchesQuery(receipt, query) {
			continue
		}
		matched++
		if len(rows) < limit {
			rows = append(rows, bankReceiptRowFrom(receipt, itemsByReceipt[receipt.ID]))
			shown = append(shown, bankReceiptRef(receipt.ID))
		}
	}

	return &bankReceiptSearch{
		SearchOutcome: querytool.SearchResult(criteria, rows, matched),
		outside:       shown,
	}, nil
}

// bankReceiptSearch names each receipt it shows, so the run's taint says which
// receipts' payer-written memos it read.
type bankReceiptSearch struct {
	querytool.SearchOutcome

	outside []agentrecord.RecordRef
}

func (s *bankReceiptSearch) TaintedRecords() []agentrecord.RecordRef { return s.outside }

func bankReceiptRef(id pulid.ID) agentrecord.RecordRef {
	return agentrecord.RecordRef{EntityType: agent.TaintEntityBankReceipt, ID: id.String()}
}

func receiptMatchesQuery(receipt *bankreceipt.BankReceipt, query string) bool {
	if query == "" {
		return true
	}

	for _, field := range []string{receipt.ReferenceNumber, receipt.Memo, receipt.ExceptionReason} {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}

// bankReceiptDetailRow is one receipt with the payments that might be it.
type bankReceiptDetailRow struct {
	bankReceiptRow

	Suggestions []matchSuggestionRow `json:"suggestions"`
	Note        string               `json:"note,omitempty"`

	outside []agentrecord.RecordRef
}

func (r *bankReceiptDetailRow) TaintedRecords() []agentrecord.RecordRef { return r.outside }

type matchSuggestionRow struct {
	CustomerPaymentID string `json:"customerPaymentId"`
	CustomerID        string `json:"customerId,omitempty"`
	ReferenceNumber   string `json:"referenceNumber,omitempty"`
	Amount            string `json:"amount"`
	Score             int    `json:"score"`
	Reason            string `json:"reason"`
}

type getBankReceiptTool struct {
	receipts bankReceiptReader
	items    workItemReader
}

func NewGetBankReceiptTool(
	receipts bankReceiptReader,
	items workItemReader,
) agenttoolports.AgentQueryTool {
	return &getBankReceiptTool{receipts: receipts, items: items}
}

func (t *getBankReceiptTool) Name() string { return "receivable_get_bank_receipt" }

func (t *getBankReceiptTool) Description() string {
	return "Read one bank receipt with the customer payments that might be it, each scored " +
		"out of 100, and its reconciliation queue entry. The score says how well a " +
		"payment's reference, amount and date agree with the receipt. A receipt in Exception has no match " +
		"yet; a score of 90 or more with a clear lead is what the system would have " +
		"matched on its own. Use the id from receivable_list_bank_receipt_exceptions or the run's subject."
}

func (t *getBankReceiptTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"bankReceiptId": agenttoolschema.RecordIDText(permission.ResourceBankReceipt,
				"The bank receipt's id, from receivable_list_bank_receipt_exceptions or "+
					"this run's subject."),
		},
		toolschema.KeyRequired:             []string{"bankReceiptId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *getBankReceiptTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceBankReceipt,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceBankReceipt,
		Rationale: "Reads a bank receipt whose memo the payer wrote; nothing changes and nothing " +
			"is sent.",
	})
}

func (t *getBankReceiptTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	id, err := querytool.RequirePulid(params.Params, "bankReceiptId")
	if err != nil {
		return nil, err
	}

	tenant := querytool.TenantOf(params)
	req := &bankreceiptports.GetBankReceiptRequest{ReceiptID: id, TenantInfo: tenant}
	receipt, err := t.receipts.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	item, err := activeWorkItem(ctx, t.items, tenant, id)
	if err != nil && !errors.Is(err, errNoActiveWorkItem) {
		return nil, err
	}

	row := &bankReceiptDetailRow{
		bankReceiptRow: bankReceiptRowFrom(receipt, item),
		Suggestions:    []matchSuggestionRow{},
		outside:        []agentrecord.RecordRef{bankReceiptRef(receipt.ID)},
	}
	if receipt.Status == bankreceipt.StatusMatched {
		row.Note = "This receipt is already matched; nothing further is needed."

		return row, nil
	}

	suggestions, err := t.receipts.SuggestMatches(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, suggestion := range suggestions {
		if suggestion == nil {
			continue
		}
		row.Suggestions = append(row.Suggestions, matchSuggestionRow{
			CustomerPaymentID: suggestion.CustomerPaymentID.String(),
			CustomerID:        querytool.PulidString(suggestion.CustomerID),
			ReferenceNumber:   suggestion.ReferenceNumber,
			Amount:            money.DecimalFromMinor(suggestion.AmountMinor).StringFixed(2),
			Score:             suggestion.Score,
			Reason:            suggestion.Reason,
		})
	}
	if len(row.Suggestions) == 0 {
		row.Note = "No posted customer payment resembles this receipt. Search receivable_list_customer_payments " +
			"by the reference or memo, or identify the customer and the invoices it pays."
	}

	return row, nil
}
