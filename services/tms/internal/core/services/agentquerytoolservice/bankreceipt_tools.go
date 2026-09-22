package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
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
	Get(ctx context.Context, req *serviceports.GetBankReceiptRequest) (*bankreceipt.BankReceipt, error)
	ListExceptions(ctx context.Context, tenant pagination.TenantInfo) ([]*bankreceipt.BankReceipt, error)
	SuggestMatches(
		ctx context.Context,
		req *serviceports.GetBankReceiptRequest,
	) ([]*serviceports.BankReceiptMatchSuggestion, error)
}

// workItemReader reads the reconciliation queue entry behind a receipt.
type workItemReader interface {
	ListActive(ctx context.Context, tenant pagination.TenantInfo) ([]*bankreceiptworkitem.WorkItem, error)
	GetActiveByReceiptID(
		ctx context.Context,
		tenant pagination.TenantInfo,
		receiptID pulid.ID,
	) (*bankreceiptworkitem.WorkItem, error)
}

type customerPaymentLister interface {
	List(
		ctx context.Context,
		req *repositories.ListCustomerPaymentsRequest,
	) (*pagination.ListResult[*customerpayment.Payment], error)
}

// bankReceiptRow is a receipt in the words the model needs: when the money
// arrived, how much, what the bank said about it, and why it is waiting.
type bankReceiptRow struct {
	ID                       string       `json:"id"`
	ReceiptDate              optionalDate `json:"receiptDate"`
	Amount                   string       `json:"amount"`
	ReferenceNumber          string       `json:"referenceNumber,omitempty"`
	Memo                     string       `json:"memo,omitempty"`
	Status                   string       `json:"status"`
	ExceptionReason          string       `json:"exceptionReason,omitempty"`
	MatchedCustomerPaymentID string       `json:"matchedCustomerPaymentId,omitempty"`
	MatchedOn                optionalDate `json:"matchedOn"`
	ImportBatchID            string       `json:"importBatchId,omitempty"`
	WorkItem                 *workItemRow `json:"workItem,omitempty"`
}

type workItemRow struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	AssignedToUserID string `json:"assignedToUserId,omitempty"`
	ResolutionType   string `json:"resolutionType,omitempty"`
	ResolutionNote   string `json:"resolutionNote,omitempty"`
}

func bankReceiptRowFrom(entity *bankreceipt.BankReceipt, item *bankreceiptworkitem.WorkItem) bankReceiptRow {
	row := bankReceiptRow{
		ID:              entity.ID.String(),
		ReceiptDate:     recordedDate(entity.ReceiptDate),
		Amount:          money.DecimalFromMinor(entity.AmountMinor).StringFixed(2),
		ReferenceNumber: entity.ReferenceNumber,
		Memo:            entity.Memo,
		Status:          string(entity.Status),
		ExceptionReason: entity.ExceptionReason,
		MatchedOn:       expectedDate(pointerSeconds(entity.MatchedAt), "not matched"),
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

// activeWorkItem reads the queue entry for a receipt, and reads "none" as
// none rather than as a failure: a receipt with no open item is the normal
// state of a matched one.
func activeWorkItem(
	ctx context.Context,
	items workItemReader,
	tenant pagination.TenantInfo,
	receiptID pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	if items == nil {
		return nil, nil
	}

	item, err := items.GetActiveByReceiptID(ctx, tenant, receiptID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return item, nil
}

type listBankReceiptExceptionsTool struct {
	receipts bankReceiptReader
	items    workItemReader
}

func newListBankReceiptExceptionsTool(
	receipts bankReceiptReader,
	items workItemReader,
) serviceports.AgentQueryTool {
	return &listBankReceiptExceptionsTool{receipts: receipts, items: items}
}

func (t *listBankReceiptExceptionsTool) Name() string { return "list_bank_receipt_exceptions" }

func (t *listBankReceiptExceptionsTool) Description() string {
	return "List the bank receipts that could not be matched to a customer payment on " +
		"their own and are waiting in the reconciliation queue, each with its amount, " +
		"the bank's reference and memo, why it was not matched, and the queue entry with " +
		"who it is assigned to. Narrow with query to match the reference, memo or reason. " +
		"Use get_bank_receipt for one receipt's candidate payments."
}

func (t *listBankReceiptExceptionsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Words to look for in the reference, the memo or the exception reason.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many to return, at most %d.", maxBankReceiptRows),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listBankReceiptExceptionsTool) PermissionResource() permission.Resource {
	return permission.ResourceBankReceipt
}

func (t *listBankReceiptExceptionsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultBankReceiptRows)
	if limit <= 0 || limit > maxBankReceiptRows {
		limit = maxBankReceiptRows
	}
	query := strings.ToLower(strings.TrimSpace(optionalString(params.Params, "query")))

	criteria := filtercatalog.NewCriteria("unmatched bank receipts").At(clockFor(params))
	criteria.Text(query)

	tenant := tenantOf(params)
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
	matched := 0
	for _, receipt := range receipts {
		if receipt == nil || !receiptMatchesQuery(receipt, query) {
			continue
		}
		matched++
		if len(rows) < limit {
			rows = append(rows, bankReceiptRowFrom(receipt, itemsByReceipt[receipt.ID]))
		}
	}

	return searchResult(criteria, rows, matched), nil
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
}

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

func newGetBankReceiptTool(receipts bankReceiptReader, items workItemReader) serviceports.AgentQueryTool {
	return &getBankReceiptTool{receipts: receipts, items: items}
}

func (t *getBankReceiptTool) Name() string { return "get_bank_receipt" }

func (t *getBankReceiptTool) Description() string {
	return "Read one bank receipt with the customer payments that might be it, each scored " +
		"out of 100 on how well its reference, amount and date agree with the receipt, " +
		"and the reconciliation queue entry for it. A receipt in Exception has no match " +
		"yet; a score of 90 or more with a clear lead is what the system would have " +
		"matched on its own. Use the id from list_bank_receipt_exceptions or the run's subject."
}

func (t *getBankReceiptTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"bankReceiptId": map[string]any{
				"type":        "string",
				"description": "The bank receipt's id.",
			},
		},
		"required":             []string{"bankReceiptId"},
		"additionalProperties": false,
	}
}

func (t *getBankReceiptTool) PermissionResource() permission.Resource {
	return permission.ResourceBankReceipt
}

func (t *getBankReceiptTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "bankReceiptId")
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	req := &serviceports.GetBankReceiptRequest{ReceiptID: id, TenantInfo: tenant}
	receipt, err := t.receipts.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	item, err := activeWorkItem(ctx, t.items, tenant, id)
	if err != nil {
		return nil, err
	}

	row := bankReceiptDetailRow{
		bankReceiptRow: bankReceiptRowFrom(receipt, item),
		Suggestions:    []matchSuggestionRow{},
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
			CustomerID:        pulidString(suggestion.CustomerID),
			ReferenceNumber:   suggestion.ReferenceNumber,
			Amount:            money.DecimalFromMinor(suggestion.AmountMinor).StringFixed(2),
			Score:             suggestion.Score,
			Reason:            suggestion.Reason,
		})
	}
	if len(row.Suggestions) == 0 {
		row.Note = "No posted customer payment resembles this receipt. Search list_customer_payments " +
			"by the reference or memo, or identify the customer and the invoices it pays."
	}

	return row, nil
}

func pulidString(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

// customerPaymentRow is a posted payment in the words the model needs to
// tell whether it is the one a receipt represents.
type customerPaymentRow struct {
	ID              string                  `json:"id"`
	CustomerID      string                  `json:"customerId"`
	PaymentDate     optionalDate            `json:"paymentDate"`
	Amount          string                  `json:"amount"`
	AppliedAmount   string                  `json:"appliedAmount"`
	UnappliedAmount string                  `json:"unappliedAmount"`
	Currency        string                  `json:"currency"`
	Method          string                  `json:"method"`
	ReferenceNumber string                  `json:"referenceNumber,omitempty"`
	Memo            string                  `json:"memo,omitempty"`
	Status          string                  `json:"status"`
	Applications    []paymentApplicationRow `json:"applications,omitempty"`
}

type paymentApplicationRow struct {
	InvoiceID      string `json:"invoiceId"`
	Amount         string `json:"amount"`
	ShortPayAmount string `json:"shortPayAmount,omitempty"`
}

func customerPaymentRowFrom(payment *customerpayment.Payment) customerPaymentRow {
	row := customerPaymentRow{
		ID:              payment.ID.String(),
		CustomerID:      pulidString(payment.CustomerID),
		PaymentDate:     recordedDate(payment.PaymentDate),
		Amount:          money.DecimalFromMinor(payment.AmountMinor).StringFixed(2),
		AppliedAmount:   money.DecimalFromMinor(payment.AppliedAmountMinor).StringFixed(2),
		UnappliedAmount: money.DecimalFromMinor(payment.UnappliedAmountMinor).StringFixed(2),
		Currency:        payment.CurrencyCode,
		Method:          string(payment.PaymentMethod),
		ReferenceNumber: payment.ReferenceNumber,
		Memo:            payment.Memo,
		Status:          string(payment.Status),
	}
	if row.Currency == "" {
		row.Currency = money.DefaultCurrencyCode
	}
	if len(payment.Applications) > 0 {
		row.Applications = make([]paymentApplicationRow, 0, len(payment.Applications))
		for _, application := range payment.Applications {
			if application == nil {
				continue
			}
			entry := paymentApplicationRow{
				InvoiceID: application.InvoiceID.String(),
				Amount:    money.DecimalFromMinor(application.AppliedAmountMinor).StringFixed(2),
			}
			if application.ShortPayAmountMinor > 0 {
				entry.ShortPayAmount = money.DecimalFromMinor(application.ShortPayAmountMinor).StringFixed(2)
			}
			row.Applications = append(row.Applications, entry)
		}
	}

	return row
}

type listCustomerPaymentsTool struct {
	payments customerPaymentLister
}

func newListCustomerPaymentsTool(payments customerPaymentLister) serviceports.AgentQueryTool {
	return &listCustomerPaymentsTool{payments: payments}
}

func (t *listCustomerPaymentsTool) Name() string { return "list_customer_payments" }

func (t *listCustomerPaymentsTool) Description() string {
	return "List recorded customer payments, newest first, with the amount, how much of " +
		"it is applied to invoices, the method, the reference and the memo. Search by " +
		"reference or memo text, narrow to one customer, or include reversed payments. " +
		"Use it to find the payment a bank receipt represents when get_bank_receipt " +
		"suggested nothing, and to see whether a customer's payment is already recorded " +
		"before posting another."
}

func (t *listCustomerPaymentsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Words to look for in the reference number or the memo.",
			},
			"customerId": map[string]any{"type": "string"},
			"status": map[string]any{
				"type":        "string",
				"enum":        []string{string(customerpayment.StatusPosted), string(customerpayment.StatusReversed)},
				"description": "Which payments to list. Defaults to Posted.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many to return, at most %d.", maxBankReceiptRows),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listCustomerPaymentsTool) PermissionResource() permission.Resource {
	return permission.ResourceCustomerPayment
}

func (t *listCustomerPaymentsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultBankReceiptRows)
	if limit <= 0 || limit > maxBankReceiptRows {
		limit = maxBankReceiptRows
	}

	criteria := filtercatalog.NewCriteria("customer payments").At(clockFor(params))

	req := &repositories.ListCustomerPaymentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: limit},
			Query:      strings.TrimSpace(optionalString(params.Params, "query")),
		},
		Status: customerpayment.StatusPosted,
	}
	criteria.Text(req.Filter.Query)

	if raw := optionalString(params.Params, "customerId"); raw != "" {
		customerID, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("customerId %q is not a record id", raw)
		}
		req.CustomerID = customerID
		criteria.Field("customer", raw)
	}

	if raw := optionalString(params.Params, "status"); raw != "" {
		status := customerpayment.Status(raw)
		if status != customerpayment.StatusPosted && status != customerpayment.StatusReversed {
			return nil, fmt.Errorf("status %q is not Posted or Reversed", raw)
		}
		req.Status = status
	}
	criteria.Field("status", string(req.Status))

	result, err := t.payments.List(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]customerPaymentRow, 0, len(result.Items))
	for _, payment := range result.Items {
		if payment != nil {
			rows = append(rows, customerPaymentRowFrom(payment))
		}
	}

	return searchResult(criteria, rows, result.Total), nil
}
