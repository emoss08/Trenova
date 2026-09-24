package agentquerytoolservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultAccountingRows = 25
	maxAccountingRows     = 100
	maxStatementRows      = 50
	maxJournalLines       = 100
	maxCloseBlockers      = 50

	bucketCurrent = "current"
	bucket1To30   = "1-30"
	bucket31To60  = "31-60"
	bucket61To90  = "61-90"
	bucketOver90  = "over 90"
)

var journalStatuses = []string{
	string(journalentry.StatusDraft),
	string(journalentry.StatusPending),
	string(journalentry.StatusApproved),
	string(journalentry.StatusPosted),
	string(journalentry.StatusReversed),
	string(journalentry.StatusRejected),
	string(journalentry.StatusVoid),
}

func accountingToolProviders() []any {
	return []any{
		provideGetARAgingTool,
		provideListAROpenItemsTool,
		provideGetCustomerStatementTool,
		provideListCollectionsWorklistTool,
		newListJournalEntriesTool,
		newGetJournalEntryTool,
		newListGLAccountsTool,
		newListFiscalPeriodsTool,
		provideGetFiscalCloseBlockersTool,
	}
}

type receivablesReader interface {
	GetAgingSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		asOfDate int64,
	) (*serviceports.ARAgingSummary, error)
	ListOpenItems(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		asOfDate int64,
	) ([]*repositories.AROpenItem, error)
	GetCustomerStatement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		startDate, asOfDate int64,
	) (*serviceports.ARCustomerStatement, error)
	GetCollectionsWorklist(
		ctx context.Context,
		req repositories.ListARCollectionsWorklistRequest,
	) ([]*repositories.ARCollectionsWorklistItem, error)
}

func provideGetARAgingTool(
	receivables *accountsreceivableservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetARAgingTool(receivables, permissions)
}

func provideListAROpenItemsTool(
	receivables *accountsreceivableservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListAROpenItemsTool(receivables, permissions)
}

func provideGetCustomerStatementTool(
	receivables *accountsreceivableservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetCustomerStatementTool(receivables, permissions)
}

func provideListCollectionsWorklistTool(
	receivables *accountsreceivableservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListCollectionsWorklistTool(receivables, permissions)
}

func provideGetFiscalCloseBlockersTool(
	periods *fiscalperiodservice.Service,
) serviceports.AgentQueryTool {
	return newGetFiscalCloseBlockersTool(periods)
}

func minorText(minor int64) string {
	return money.DecimalFromMinor(minor).StringFixed(2)
}

type arBuckets struct {
	Current    string `json:"current,omitempty"`
	Days1To30  string `json:"days1To30,omitempty"`
	Days31To60 string `json:"days31To60,omitempty"`
	Days61To90 string `json:"days61To90,omitempty"`
	Over90     string `json:"over90,omitempty"`
	TotalOpen  string `json:"totalOpen,omitempty"`
}

func arBucketsFrom(totals repositories.ARAgingBucketTotals) *arBuckets {
	return &arBuckets{
		Current:    minorText(totals.CurrentMinor),
		Days1To30:  minorText(totals.Days1To30Minor),
		Days31To60: minorText(totals.Days31To60Minor),
		Days61To90: minorText(totals.Days61To90Minor),
		Over90:     minorText(totals.DaysOver90Minor),
		TotalOpen:  minorText(totals.TotalOpenMinor),
	}
}

func oldestBucket(totals repositories.ARAgingBucketTotals) (string, int) {
	switch {
	case totals.DaysOver90Minor > 0:
		return bucketOver90, 4
	case totals.Days61To90Minor > 0:
		return bucket61To90, 3
	case totals.Days31To60Minor > 0:
		return bucket31To60, 2
	case totals.Days1To30Minor > 0:
		return bucket1To30, 1
	default:
		return bucketCurrent, 0
	}
}

type getARAgingTool struct {
	receivables receivablesReader
	access      fieldAccess
}

func newGetARAgingTool(
	receivables receivablesReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getARAgingTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *getARAgingTool) Name() string { return "get_ar_aging" }

func (t *getARAgingTool) Description() string {
	return "Summarize accounts receivable aging: which customers owe money and how overdue " +
		"it is, in current, 1-30, 31-60, 61-90 and over-90-day buckets. Each row names the " +
		"oldest bucket holding a balance. Use list_ar_open_items for the invoices behind a " +
		"customer's balance and get_customer_statement for their history. Amounts are " +
		"named in withheldByAccess when your data access does not reach them."
}

func (t *getARAgingTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramAsOf: dateParam("The day to age as of. Defaults to today."),
		paramCustomerID: stringParam("Only this customer, by id from list_customers or a " +
			"row of this tool."),
		"overdueOnly": boolParam("Only customers with a balance past due."),
	}, defaultAccountingRows, maxAccountingRows))
}

func (t *getARAgingTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountsReceivable})
}

type arAgingRow struct {
	CustomerID   string `json:"customerId"`
	Customer     string `json:"customer"`
	OldestBucket string `json:"oldestBucket"`
	Current      string `json:"current,omitempty"`
	Days1To30    string `json:"days1To30,omitempty"`
	Days31To60   string `json:"days31To60,omitempty"`
	Days61To90   string `json:"days61To90,omitempty"`
	Over90       string `json:"over90,omitempty"`
	TotalOpen    string `json:"totalOpen,omitempty"`

	rank int
	open int64
}

type arAgingOutcome struct {
	*gatedOutcome

	AsOf   optionalDate `json:"asOf"`
	Totals *arBuckets   `json:"totals,omitempty"`
}

func (t *getARAgingTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	clk := clockFor(params)
	asOf, err := readAsOf(params.Params, clk)
	if err != nil {
		return nil, err
	}
	customerID, err := optionalID(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	overdueOnly := optionalBool(params.Params, "overdueOnly")
	window := readPage(params.Params, defaultAccountingRows, maxAccountingRows)

	criteria := filtercatalog.NewCriteria("customers with open receivables").At(clk)
	if customerID.IsNotNil() {
		criteria.Field(labelCustomer, customerID.String())
	}
	if overdueOnly {
		criteria.Field("limited to", "past due balances")
	}

	summary, err := t.receivables.GetAgingSummary(ctx, tenantOf(params), asOf)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceAccountsReceivable)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)
	rows := agingRows(summary.Rows, agingFilter{
		customerID:  customerID,
		overdueOnly: overdueOnly,
		showAmounts: showAmounts,
	})

	matched := len(rows)
	page, more := slicePage(window, rows)
	found := searchResult(criteria, page, matched).paged(window, more)
	outcome := arAgingOutcome{
		gatedOutcome: gatedResult(&found, gate),
		AsOf:         recordedDate(summary.AsOfDate),
	}
	if showAmounts && customerID.IsNil() {
		outcome.Totals = arBucketsFrom(summary.Totals)
	}

	return outcome, nil
}

func readAsOf(params map[string]any, clk clock) (int64, error) {
	asOf, err := readDay(params, paramAsOf, clk)
	if err != nil || asOf == 0 {
		return asOf, err
	}

	return endOfDay(clk, asOf), nil
}

type agingFilter struct {
	customerID  pulid.ID
	overdueOnly bool
	showAmounts bool
}

func agingRows(entries []*repositories.ARCustomerAgingRow, filter agingFilter) []arAgingRow {
	rows := make([]arAgingRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Buckets.TotalOpenMinor == 0 {
			continue
		}
		if filter.customerID.IsNotNil() && entry.CustomerID != filter.customerID {
			continue
		}
		bucket, rank := oldestBucket(entry.Buckets)
		if filter.overdueOnly && rank == 0 {
			continue
		}
		row := arAgingRow{
			CustomerID:   entry.CustomerID.String(),
			Customer:     entry.CustomerName,
			OldestBucket: bucket,
			rank:         rank,
			open:         entry.Buckets.TotalOpenMinor,
		}
		if filter.showAmounts {
			buckets := arBucketsFrom(entry.Buckets)
			row.Current = buckets.Current
			row.Days1To30 = buckets.Days1To30
			row.Days31To60 = buckets.Days31To60
			row.Days61To90 = buckets.Days61To90
			row.Over90 = buckets.Over90
			row.TotalOpen = buckets.TotalOpen
		}
		rows = append(rows, row)
	}

	slices.SortStableFunc(rows, func(a, b arAgingRow) int {
		if byRank := cmp.Compare(b.rank, a.rank); byRank != 0 {
			return byRank
		}
		if filter.showAmounts {
			if byOpen := cmp.Compare(b.open, a.open); byOpen != 0 {
				return byOpen
			}
		}

		return strings.Compare(strings.ToLower(a.Customer), strings.ToLower(b.Customer))
	})

	return rows
}

type listAROpenItemsTool struct {
	receivables receivablesReader
	access      fieldAccess
}

func newListAROpenItemsTool(
	receivables receivablesReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listAROpenItemsTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *listAROpenItemsTool) Name() string { return "list_ar_open_items" }

func (t *listAROpenItemsTool) Description() string {
	return "List the open receivables, the unpaid or partly paid invoices customers still " +
		"owe, most overdue first. Each row has the due date, days past due, dispute and " +
		"short-pay state, and the open balance. Narrow to one customer, a minimum number " +
		"of days past due, or disputed invoices. get_invoice expands one."
}

func (t *listAROpenItemsTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramCustomerID: stringParam("Only this customer's invoices, by id from list_customers " +
			"or get_ar_aging."),
		"minDaysPastDue": intParam("Only invoices at least this many days past due."),
		"disputedOnly":   boolParam("Only invoices the customer disputes."),
		paramAsOf:        dateParam("The day to age as of. Defaults to today."),
	}, defaultAccountingRows, maxAccountingRows))
}

func (t *listAROpenItemsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountsReceivable})
}

type arOpenItemRow struct {
	InvoiceID        string       `json:"invoiceId"`
	InvoiceNumber    string       `json:"invoiceNumber"`
	CustomerID       string       `json:"customerId"`
	Customer         string       `json:"customer"`
	BillType         string       `json:"billType,omitempty"`
	ProNumber        string       `json:"proNumber,omitempty"`
	InvoiceDate      optionalDate `json:"invoiceDate"`
	DueDate          optionalDate `json:"dueDate"`
	DaysPastDue      int          `json:"daysPastDue"`
	SettlementStatus string       `json:"settlementStatus"`
	DisputeStatus    string       `json:"disputeStatus,omitempty"`
	HasShortPay      bool         `json:"hasShortPay"`
	Currency         string       `json:"currency"`
	Total            string       `json:"totalAmount,omitempty"`
	Applied          string       `json:"appliedAmount,omitempty"`
	Open             string       `json:"openAmount,omitempty"`
}

func arOpenItemRowFrom(item *repositories.AROpenItem, showAmounts bool) arOpenItemRow {
	row := arOpenItemRow{
		InvoiceID:        item.InvoiceID.String(),
		InvoiceNumber:    item.InvoiceNumber,
		CustomerID:       item.CustomerID.String(),
		Customer:         item.CustomerName,
		BillType:         item.BillType,
		ProNumber:        item.ShipmentProNumber,
		InvoiceDate:      recordedDate(item.InvoiceDate),
		DueDate:          recordedDate(item.DueDate),
		DaysPastDue:      max(item.DaysPastDue, 0),
		SettlementStatus: item.SettlementStatus,
		DisputeStatus:    item.DisputeStatus,
		HasShortPay:      item.HasShortPay,
		Currency:         money.CurrencyCode(item.CurrencyCode),
	}
	if showAmounts {
		row.Total = minorText(item.TotalAmountMinor)
		row.Applied = minorText(item.AppliedAmountMinor)
		row.Open = minorText(item.OpenAmountMinor)
	}

	return row
}

func (t *listAROpenItemsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	clk := clockFor(params)
	asOf, err := readAsOf(params.Params, clk)
	if err != nil {
		return nil, err
	}
	customerID, err := optionalID(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	minDays := max(optionalInt(params.Params, "minDaysPastDue", 0), 0)
	disputedOnly := optionalBool(params.Params, "disputedOnly")
	window := readPage(params.Params, defaultAccountingRows, maxAccountingRows)

	criteria := filtercatalog.NewCriteria("open receivables").At(clk)
	if customerID.IsNotNil() {
		criteria.Field(labelCustomer, customerID.String())
	}
	if minDays > 0 {
		criteria.Field("at least days past due", strconv.Itoa(minDays))
	}
	if disputedOnly {
		criteria.Field("limited to", "disputed invoices")
	}

	items, err := t.receivables.ListOpenItems(ctx, tenantOf(params), customerID, asOf)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceAccountsReceivable)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)

	kept := make([]*repositories.AROpenItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.DaysPastDue < minDays {
			continue
		}
		if disputedOnly && !strings.EqualFold(item.DisputeStatus, "Disputed") {
			continue
		}
		kept = append(kept, item)
	}
	slices.SortStableFunc(kept, func(a, b *repositories.AROpenItem) int {
		if byDays := cmp.Compare(b.DaysPastDue, a.DaysPastDue); byDays != 0 {
			return byDays
		}

		return cmp.Compare(a.DueDate, b.DueDate)
	})

	page, more := slicePage(window, kept)
	rows := make([]arOpenItemRow, 0, len(page))
	for _, item := range page {
		rows = append(rows, arOpenItemRowFrom(item, showAmounts))
	}

	found := searchResult(criteria, rows, len(kept)).paged(window, more)

	return gatedResult(&found, gate), nil
}

type getCustomerStatementTool struct {
	receivables receivablesReader
	access      fieldAccess
}

func newGetCustomerStatementTool(
	receivables receivablesReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getCustomerStatementTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *getCustomerStatementTool) Name() string { return "get_customer_statement" }

func (t *getCustomerStatementTool) Description() string {
	return "Build one customer's account statement: invoices, payments and credits over a " +
		"period with the running balance, what is still open, and their aging. It answers " +
		"what a customer was billed and paid and what they owe now. Take the customer's " +
		"id from list_customers or get_ar_aging."
}

func (t *getCustomerStatementTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramCustomerID: stringParam("The customer's id, from list_customers, get_ar_aging or " +
			onThePage),
		paramStartDate: dateParam("The first day of the statement period. Omit for the " +
			"customer's whole history."),
		paramAsOf: dateParam("The statement date. Defaults to today."),
	}, paramCustomerID)
}

func (t *getCustomerStatementTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountsReceivable})
}

type statementTransactionRow struct {
	Date           optionalDate `json:"date"`
	EventType      string       `json:"eventType"`
	DocumentNumber string       `json:"documentNumber,omitempty"`
	Charge         string       `json:"charge,omitempty"`
	Payment        string       `json:"payment,omitempty"`
	RunningBalance string       `json:"runningBalance,omitempty"`
}

type customerStatementView struct {
	CustomerID          string                    `json:"customerId"`
	Customer            string                    `json:"customer"`
	StatementDate       optionalDate              `json:"statementDate"`
	StartDate           optionalDate              `json:"startDate"`
	OpeningBalance      string                    `json:"openingBalance,omitempty"`
	TotalCharges        string                    `json:"totalCharges,omitempty"`
	TotalPayments       string                    `json:"totalPayments,omitempty"`
	EndingBalance       string                    `json:"endingBalance,omitempty"`
	Aging               *arBuckets                `json:"aging,omitempty"`
	OldestBucket        string                    `json:"oldestBucket"`
	TransactionCount    int                       `json:"transactionCount"`
	Transactions        []statementTransactionRow `json:"transactions"`
	TransactionsOmitted int                       `json:"earlierTransactionsOmitted,omitempty"`
	OpenItemCount       int                       `json:"openItemCount"`
	OpenItems           []arOpenItemRow           `json:"openItems"`
	OpenItemsOmitted    int                       `json:"openItemsOmitted,omitempty"`
	Withheld            []string                  `json:"withheldByAccess,omitempty"`
}

func (t *getCustomerStatementTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	customerID, err := requirePulid(params.Params, paramCustomerID)
	if err != nil {
		return nil, err
	}
	clk := clockFor(params)
	start, err := readDay(params.Params, paramStartDate, clk)
	if err != nil {
		return nil, err
	}
	asOf, err := readAsOf(params.Params, clk)
	if err != nil {
		return nil, err
	}
	if start > 0 && asOf > 0 && start > asOf {
		return nil, errors.New("startDate must be on or before asOf")
	}

	statement, err := t.receivables.GetCustomerStatement(
		ctx, tenantOf(params), customerID, start, asOf,
	)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceAccountsReceivable)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)

	bucket, _ := oldestBucket(statement.Aging)
	view := customerStatementView{
		CustomerID:       statement.CustomerID.String(),
		Customer:         statement.CustomerName,
		StatementDate:    recordedDate(statement.StatementDate),
		StartDate:        expectedDate(statement.StartDate, "whole history"),
		OldestBucket:     bucket,
		TransactionCount: len(statement.Transactions),
		OpenItemCount:    len(statement.OpenItems),
	}
	if showAmounts {
		view.OpeningBalance = minorText(statement.OpeningBalanceMinor)
		view.TotalCharges = minorText(statement.TotalChargesMinor)
		view.TotalPayments = minorText(statement.TotalPaymentsMinor)
		view.EndingBalance = minorText(statement.EndingBalanceMinor)
		view.Aging = arBucketsFrom(statement.Aging)
	}
	view.Transactions, view.TransactionsOmitted = statementTransactions(
		statement.Transactions, showAmounts,
	)
	view.OpenItems, view.OpenItemsOmitted = statementOpenItems(statement.OpenItems, showAmounts)
	view.Withheld = gate.Withheld()

	return view, nil
}

func statementTransactions(
	transactions []*serviceports.ARStatementTransaction,
	showAmounts bool,
) ([]statementTransactionRow, int) {
	omitted := 0
	if len(transactions) > maxStatementRows {
		omitted = len(transactions) - maxStatementRows
		transactions = transactions[omitted:]
	}

	rows := make([]statementTransactionRow, 0, len(transactions))
	for _, txn := range transactions {
		if txn == nil {
			continue
		}
		row := statementTransactionRow{
			Date:           recordedDate(txn.TransactionDate),
			EventType:      txn.EventType,
			DocumentNumber: txn.DocumentNumber,
		}
		if showAmounts {
			if txn.ChargeMinor != 0 {
				row.Charge = minorText(txn.ChargeMinor)
			}
			if txn.PaymentMinor != 0 {
				row.Payment = minorText(txn.PaymentMinor)
			}
			row.RunningBalance = minorText(txn.RunningBalanceMinor)
		}
		rows = append(rows, row)
	}

	return rows, omitted
}

func statementOpenItems(
	items []*repositories.AROpenItem,
	showAmounts bool,
) ([]arOpenItemRow, int) {
	omitted := 0
	if len(items) > maxStatementRows {
		omitted = len(items) - maxStatementRows
		items = items[:maxStatementRows]
	}

	rows := make([]arOpenItemRow, 0, len(items))
	for _, item := range items {
		if item != nil {
			rows = append(rows, arOpenItemRowFrom(item, showAmounts))
		}
	}

	return rows, omitted
}

type listCollectionsWorklistTool struct {
	receivables receivablesReader
	access      fieldAccess
}

func newListCollectionsWorklistTool(
	receivables receivablesReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listCollectionsWorklistTool{
		receivables: receivables,
		access:      newFieldAccess(permissions),
	}
}

func (t *listCollectionsWorklistTool) Name() string { return "list_collections_worklist" }

func (t *listCollectionsWorklistTool) Description() string {
	return "List the collections worklist: overdue and disputed invoices ranked for a " +
		"collector to chase, each marked Critical, Warning or Watch. Rows carry the " +
		"customer, days past due, the open dispute's reason and the open amount. Use " +
		"get_customer_statement before calling a customer about their balance."
}

func (t *listCollectionsWorklistTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramAsOf: dateParam("The day to rank as of. Defaults to today."),
		paramSeverity: enumParam("Only rows of this severity.",
			[]string{"Critical", "Warning", "Watch"}),
		paramLimit: intParam(fmt.Sprintf("How many rows to return: %d unless you ask, at most %d.",
			defaultAccountingRows, maxAccountingRows)),
	})
}

func (t *listCollectionsWorklistTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceAccountsReceivable})
}

type worklistRow struct {
	InvoiceID       string       `json:"invoiceId"`
	InvoiceNumber   string       `json:"invoiceNumber"`
	CustomerID      string       `json:"customerId"`
	Customer        string       `json:"customer"`
	Severity        string       `json:"severity"`
	DueDate         optionalDate `json:"dueDate"`
	DaysPastDue     int          `json:"daysPastDue"`
	Disputed        bool         `json:"disputed"`
	HasShortPay     bool         `json:"hasShortPay"`
	DisputeReason   string       `json:"disputeReason,omitempty"`
	DisputeOpenedAt optionalDate `json:"disputeOpenedAt"`
	Open            string       `json:"openAmount,omitempty"`
	DisputedAmount  string       `json:"disputedAmount,omitempty"`
}

func (t *listCollectionsWorklistTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	clk := clockFor(params)
	asOf, err := readAsOf(params.Params, clk)
	if err != nil {
		return nil, err
	}
	limit := optionalInt(params.Params, paramLimit, defaultAccountingRows)
	if limit <= 0 {
		limit = defaultAccountingRows
	}
	limit = min(limit, maxAccountingRows)
	severity := optionalString(params.Params, paramSeverity)

	criteria := filtercatalog.NewCriteria("collections worklist items").At(clk)
	if severity != "" {
		criteria.Field(paramSeverity, severity)
	}

	fetch := limit
	if severity != "" {
		fetch = maxAccountingRows
	}
	items, err := t.receivables.GetCollectionsWorklist(ctx,
		repositories.ListARCollectionsWorklistRequest{
			TenantInfo: tenantOf(params),
			AsOfDate:   asOf,
			Limit:      fetch,
		})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceAccountsReceivable)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)

	rows := make([]worklistRow, 0, min(len(items), limit))
	matched := 0
	for _, item := range items {
		if item == nil || (severity != "" && !strings.EqualFold(item.Severity, severity)) {
			continue
		}
		matched++
		if len(rows) == limit {
			continue
		}
		row := worklistRow{
			InvoiceID:       item.InvoiceID.String(),
			InvoiceNumber:   item.InvoiceNumber,
			CustomerID:      item.CustomerID.String(),
			Customer:        item.CustomerName,
			Severity:        item.Severity,
			DueDate:         recordedDate(item.DueDate),
			DaysPastDue:     max(item.DaysPastDue, 0),
			Disputed:        item.IsDisputed,
			HasShortPay:     item.HasShortPay,
			DisputeReason:   item.OpenDisputeReasonCode,
			DisputeOpenedAt: expectedDate(derefInt64(item.DisputeOpenedAt), "no open dispute"),
		}
		if showAmounts {
			row.Open = minorText(item.OpenAmountMinor)
			if item.DisputedAmountMinor != 0 {
				row.DisputedAmount = minorText(item.DisputedAmountMinor)
			}
		}
		rows = append(rows, row)
	}

	found := searchResult(criteria, rows, matched)

	return gatedResult(&found, gate), nil
}

type journalEntryReader interface {
	List(
		ctx context.Context,
		req *repositories.ListJournalEntriesRequest,
	) (*pagination.ListResult[*journalentry.JournalEntry], error)
	GetByID(
		ctx context.Context,
		req repositories.GetJournalEntryByIDRequest,
	) (*journalentry.JournalEntry, error)
}

type glAccountLookup interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetGLAccountsByIDsRequest,
	) ([]*glaccount.GLAccount, error)
}

type listJournalEntriesTool struct {
	entries journalEntryReader
	access  fieldAccess
}

func newListJournalEntriesTool(
	entries repositories.JournalEntryRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listJournalEntriesTool{entries: entries, access: newFieldAccess(permissions)}
}

func (t *listJournalEntriesTool) Name() string { return "list_journal_entries" }

func (t *listJournalEntriesTool) Description() string {
	return "List general ledger journal entries, newest accounting date first, with their " +
		"number, type, status, source reference and debit and credit totals. Narrow by " +
		"status, source, fiscal period, an accounting date range or text in the number or " +
		"description. get_journal_entry opens one with its lines."
}

func (t *listJournalEntriesTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramQuery: stringParam("Words to look for in the entry number, description or " +
			"source reference."),
		paramStatus: enumParam("Only entries in this status.", journalStatuses),
		"referenceType": stringParam("Only entries from this source, such as Invoice, " +
			"CustomerPayment or DriverSettlement."),
		paramFiscalPeriodID: stringParam("Only entries in this fiscal period, by id from " +
			"list_fiscal_periods."),
		paramFromDate: dateParam("The earliest accounting date."),
		paramToDate:   dateParam("The latest accounting date."),
	}, defaultListLimit, maxListLimit))
}

func (t *listJournalEntriesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceJournalEntry})
}

type journalEntryRow struct {
	ID              string       `json:"id"`
	EntryNumber     string       `json:"entryNumber"`
	EntryDate       optionalDate `json:"entryDate"`
	AccountingDate  optionalDate `json:"accountingDate"`
	Type            string       `json:"entryType"`
	Status          string       `json:"status"`
	ReferenceType   string       `json:"referenceType,omitempty"`
	ReferenceNumber string       `json:"referenceNumber,omitempty"`
	Description     string       `json:"description,omitempty"`
	Reversal        bool         `json:"isReversal"`
	AutoGenerated   bool         `json:"isAutoGenerated"`
	TotalDebit      string       `json:"totalDebit,omitempty"`
	TotalCredit     string       `json:"totalCredit,omitempty"`
}

func journalEntryRowFrom(entry *journalentry.JournalEntry, gate *fieldGate) journalEntryRow {
	row := journalEntryRow{
		ID:              entry.ID.String(),
		EntryNumber:     entry.EntryNumber,
		EntryDate:       recordedDate(entry.EntryDate),
		AccountingDate:  recordedDate(entry.AccountingDate),
		Type:            string(entry.EntryType),
		Status:          string(entry.Status),
		ReferenceType:   entry.ReferenceType,
		ReferenceNumber: entry.ReferenceNumber,
		Description:     entry.Description,
		Reversal:        entry.IsReversal,
		AutoGenerated:   entry.IsAutoGenerated,
	}
	if gate.show("totalDebit", "totalDebit") {
		row.TotalDebit = minorText(entry.TotalDebit)
	}
	if gate.show("totalCredit", "totalCredit") {
		row.TotalCredit = minorText(entry.TotalCredit)
	}

	return row
}

func (t *listJournalEntriesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	clk := clockFor(params)
	from, err := readDay(params.Params, paramFromDate, clk)
	if err != nil {
		return nil, err
	}
	to, err := readDay(params.Params, paramToDate, clk)
	if err != nil {
		return nil, err
	}
	if to > 0 {
		to = endOfDay(clk, to)
	}
	if from > 0 && to > 0 && from > to {
		return nil, errors.New("fromDate must be on or before toDate")
	}
	periodID, err := optionalID(params.Params, paramFiscalPeriodID)
	if err != nil {
		return nil, err
	}
	status := optionalString(params.Params, paramStatus)
	if status != "" && !slices.Contains(journalStatuses, status) {
		return nil, fmt.Errorf("status %q is not one of %s", status,
			strings.Join(journalStatuses, ", "))
	}
	window := readPage(params.Params, defaultListLimit, maxListLimit)

	req := &repositories.ListJournalEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: window.fetch(), Offset: window.offset},
			Query:      optionalString(params.Params, paramQuery),
		},
		FiscalPeriodID:      periodID,
		ReferenceType:       optionalString(params.Params, "referenceType"),
		Status:              status,
		AccountingDateStart: from,
		AccountingDateEnd:   to,
	}

	criteria := filtercatalog.NewCriteria("journal entries").At(clk)
	criteria.Text(req.Filter.Query)
	if status != "" {
		criteria.Field(paramStatus, status)
	}
	if req.ReferenceType != "" {
		criteria.Field("source", req.ReferenceType)
	}
	if periodID.IsNotNil() {
		criteria.Field("fiscal period", periodID.String())
	}
	if from > 0 {
		criteria.Field("accounted on or after", optionalString(params.Params, paramFromDate))
	}
	if to > 0 {
		criteria.Field("accounted on or before", optionalString(params.Params, paramToDate))
	}

	result, err := t.entries.List(ctx, req)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceJournalEntry)
	rows := make([]journalEntryRow, 0, len(result.Items))
	for _, entry := range result.Items {
		if entry != nil {
			rows = append(rows, journalEntryRowFrom(entry, gate))
		}
	}
	rows, more := trim(window, rows)

	found := searchResult(criteria, rows, len(rows)).paged(window, more)

	return gatedResult(&found, gate), nil
}

type getJournalEntryTool struct {
	entries  journalEntryReader
	accounts glAccountLookup
	access   fieldAccess
}

func newGetJournalEntryTool(
	entries repositories.JournalEntryRepository,
	accounts repositories.GLAccountRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getJournalEntryTool{
		entries:  entries,
		accounts: accounts,
		access:   newFieldAccess(permissions),
	}
}

func (t *getJournalEntryTool) Name() string { return "get_journal_entry" }

func (t *getJournalEntryTool) Description() string {
	return "Retrieve one journal entry by id with its lines: the GL account each line " +
		"posts to, its debit or credit, and the entry's source, approval and reversal " +
		"history. Use list_journal_entries first when you have a number or a date."
}

func (t *getJournalEntryTool) ParamSchema() map[string]any {
	return idSchema("journalEntryId", "The journal entry's id, from list_journal_entries or "+
		onThePage)
}

func (t *getJournalEntryTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceJournalEntry})
}

type journalLineRow struct {
	LineNumber  int    `json:"lineNumber"`
	AccountID   string `json:"glAccountId"`
	AccountCode string `json:"accountCode,omitempty"`
	AccountName string `json:"accountName,omitempty"`
	Description string `json:"description,omitempty"`
	CustomerID  string `json:"customerId,omitempty"`
	Debit       string `json:"debit,omitempty"`
	Credit      string `json:"credit,omitempty"`
	Reconciled  bool   `json:"reconciled"`
}

type journalEntryView struct {
	journalEntryRow

	FiscalPeriodID  string           `json:"fiscalPeriodId"`
	ReferenceID     string           `json:"referenceId,omitempty"`
	PostedAt        optionalDate     `json:"postedAt"`
	ApprovedAt      optionalDate     `json:"approvedAt"`
	RejectedAt      optionalDate     `json:"rejectedAt"`
	ReversalOfID    string           `json:"reversalOfId,omitempty"`
	ReversedByID    string           `json:"reversedById,omitempty"`
	ReversalDate    optionalDate     `json:"reversalDate"`
	ReversalReason  string           `json:"reversalReason,omitempty"`
	RejectionReason string           `json:"rejectionReason,omitempty"`
	ApprovalNotes   string           `json:"approvalNotes,omitempty"`
	LineCount       int              `json:"lineCount"`
	Lines           []journalLineRow `json:"lines"`
	LinesTruncated  bool             `json:"linesTruncated,omitempty"`
	Withheld        []string         `json:"withheldByAccess,omitempty"`
}

func (t *getJournalEntryTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "journalEntryId")
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	entry, err := t.entries.GetByID(ctx, repositories.GetJournalEntryByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	lines := entry.Lines
	truncated := len(lines) > maxJournalLines
	if truncated {
		lines = lines[:maxJournalLines]
	}

	accounts, err := t.lineAccounts(ctx, tenant, lines)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceJournalEntry)
	view := journalEntryView{
		journalEntryRow: journalEntryRowFrom(entry, gate),
		FiscalPeriodID:  entry.FiscalPeriodID.String(),
		ReferenceID:     entry.ReferenceID,
		PostedAt:        expectedDate(derefInt64(entry.PostedAt), absentNotPosted),
		ApprovedAt:      expectedDate(derefInt64(entry.ApprovedAt), absentNotApproved),
		RejectedAt:      expectedDate(derefInt64(entry.RejectedAt), "not rejected"),
		ReversalOfID:    pulidString(entry.ReversalOfID),
		ReversedByID:    pulidString(entry.ReversedByID),
		ReversalDate:    expectedDate(derefInt64(entry.ReversalDate), "not reversed"),
		LineCount:       len(entry.Lines),
		LinesTruncated:  truncated,
		Lines:           make([]journalLineRow, 0, len(lines)),
	}
	if entry.ReversalReason != "" && gate.show("reversalReason", "reversalReason") {
		view.ReversalReason = entry.ReversalReason
	}
	if entry.RejectionReason != "" && gate.show("rejectionReason", "rejectionReason") {
		view.RejectionReason = entry.RejectionReason
	}
	if entry.ApprovalNotes != "" && gate.show("approvalNotes", "approvalNotes") {
		view.ApprovalNotes = entry.ApprovalNotes
	}

	showDebit := gate.show("debitAmount", "lines.debit")
	showCredit := gate.show("creditAmount", "lines.credit")
	for _, line := range lines {
		if line == nil {
			continue
		}
		row := journalLineRow{
			LineNumber:  int(line.LineNumber),
			AccountID:   line.GLAccountID.String(),
			Description: line.Description,
			CustomerID:  pulidString(line.CustomerID),
			Reconciled:  line.IsReconciled,
		}
		if account, ok := accounts[line.GLAccountID]; ok {
			row.AccountCode = account.AccountCode
			row.AccountName = account.Name
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

func (t *getJournalEntryTool) lineAccounts(
	ctx context.Context,
	tenant pagination.TenantInfo,
	lines []*journalentry.JournalEntryLine,
) (map[pulid.ID]*glaccount.GLAccount, error) {
	out := make(map[pulid.ID]*glaccount.GLAccount, len(lines))
	if t.accounts == nil || len(lines) == 0 {
		return out, nil
	}

	ids := make([]pulid.ID, 0, len(lines))
	seen := make(map[pulid.ID]struct{}, len(lines))
	for _, line := range lines {
		if line == nil || line.GLAccountID.IsNil() {
			continue
		}
		if _, dup := seen[line.GLAccountID]; dup {
			continue
		}
		seen[line.GLAccountID] = struct{}{}
		ids = append(ids, line.GLAccountID)
	}
	if len(ids) == 0 {
		return out, nil
	}

	accounts, err := t.accounts.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
		TenantInfo:   tenant,
		GLAccountIDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read the GL accounts the entry posts to: %w", err)
	}
	for _, account := range accounts {
		if account != nil {
			out[account.ID] = account
		}
	}

	return out, nil
}

type glAccountRow struct {
	ID             string `json:"id"`
	AccountCode    string `json:"accountCode"`
	Name           string `json:"name"`
	Type           string `json:"accountType,omitempty"`
	Category       string `json:"category,omitempty"`
	Status         string `json:"status"`
	Description    string `json:"description,omitempty"`
	ParentID       string `json:"parentId,omitempty"`
	System         bool   `json:"isSystem"`
	AllowManualJE  bool   `json:"allowManualJe"`
	CurrentBalance string `json:"currentBalance,omitempty"`
	DebitBalance   string `json:"debitBalance,omitempty"`
	CreditBalance  string `json:"creditBalance,omitempty"`
}

func newListGLAccountsTool(
	repo repositories.GLAccountRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_gl_accounts",
		entityPlural: "general ledger accounts",
		summary: "List the chart of accounts: each general ledger account's code, name, " +
			"type and status, with its balances when your data access reaches them. Use " +
			"it to name the account a journal line posts to or to find an account by code.",
		resource: permission.ResourceGeneralLedgerAccount,
		config:   querybuilder.GetFieldConfiguration((*glaccount.GLAccount)(nil)),
		fields: []listField{
			{Name: paramStatus, Kind: filterEnum, Values: statusValues},
			{Name: "accountCode", Kind: filterText, Sortable: true},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "isSystem", Kind: filterBool},
			{Name: "allowManualJe", Kind: filterBool, Note: "accepts manual journal entries"},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListGLAccountsRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			showCurrent := gate.show("currentBalance", "currentBalance")
			showDebit := gate.show("debitBalance", "debitBalance")
			showCredit := gate.show("creditBalance", "creditBalance")

			return listRows(result.Items, func(item *glaccount.GLAccount) any {
				row := glAccountRow{
					ID:            item.ID.String(),
					AccountCode:   item.AccountCode,
					Name:          item.Name,
					Status:        string(item.Status),
					Description:   item.Description,
					ParentID:      pulidString(item.ParentID),
					System:        item.IsSystem,
					AllowManualJE: item.AllowManualJE,
				}
				if item.AccountType != nil {
					row.Type = item.AccountType.Name
					row.Category = string(item.AccountType.Category)
				}
				if showCurrent {
					row.CurrentBalance = minorText(item.CurrentBalance)
				}
				if showDebit {
					row.DebitBalance = minorText(item.DebitBalance)
				}
				if showCredit {
					row.CreditBalance = minorText(item.CreditBalance)
				}

				return row
			}), nil
		},
	})
}

type fiscalPeriodRow struct {
	ID           string       `json:"id"`
	FiscalYearID string       `json:"fiscalYearId"`
	Name         string       `json:"name"`
	PeriodNumber int          `json:"periodNumber"`
	PeriodType   string       `json:"periodType"`
	Status       string       `json:"status"`
	StartDate    optionalDate `json:"startDate"`
	EndDate      optionalDate `json:"endDate"`
	Adjusting    bool         `json:"isAdjusting"`
	LockedAt     optionalDate `json:"lockedAt"`
	ClosedAt     optionalDate `json:"closedAt"`
	ReopenedAt   optionalDate `json:"reopenedAt"`
	ReopenReason string       `json:"reopenReason,omitempty"`
}

func newListFiscalPeriodsTool(
	repo repositories.FiscalPeriodRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_fiscal_periods",
		entityPlural: "fiscal periods",
		summary: "List fiscal periods, the months, quarters or adjusting periods the books " +
			"close by, with their dates and whether each is open, locked or closed. " +
			"get_fiscal_close_blockers says what stops one from closing.",
		resource: permission.ResourceFiscalPeriod,
		config:   querybuilder.GetFieldConfiguration((*fiscalperiod.FiscalPeriod)(nil)),
		fields: []listField{
			{
				Name: paramStatus,
				Kind: filterEnum,
				Values: []string{
					string(fiscalperiod.StatusInactive),
					string(fiscalperiod.StatusOpen),
					string(fiscalperiod.StatusLocked),
					string(fiscalperiod.StatusClosed),
					string(fiscalperiod.StatusPermanentlyClosed),
				},
				Note: "Open accepts postings; Locked and Closed do not",
			},
			{
				Name: "periodType",
				Kind: filterEnum,
				Values: []string{
					string(fiscalperiod.PeriodTypeMonth),
					string(fiscalperiod.PeriodTypeQuarter),
					string(fiscalperiod.PeriodTypeWeek),
					string(fiscalperiod.PeriodTypeAdjusting),
				},
			},
			{Name: "name", Kind: filterText, Sortable: true},
			{Name: "periodNumber", Kind: filterNumber, Sortable: true},
			{Name: paramStartDate, Kind: filterDate, Sortable: true},
			{Name: "endDate", Kind: filterDate, Sortable: true},
			{Name: "isAdjusting", Kind: filterBool},
		},
		access: newFieldAccess(permissions),
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListFiscalPeriodsRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *fiscalperiod.FiscalPeriod) any {
				row := fiscalPeriodRow{
					ID:           item.ID.String(),
					FiscalYearID: item.FiscalYearID.String(),
					Name:         item.Name,
					PeriodNumber: item.PeriodNumber,
					PeriodType:   string(item.PeriodType),
					Status:       string(item.Status),
					StartDate:    recordedDate(item.StartDate),
					EndDate:      recordedDate(item.EndDate),
					Adjusting:    item.IsAdjusting,
					LockedAt:     expectedDate(derefInt64(item.LockedAt), "not locked"),
					ClosedAt:     expectedDate(derefInt64(item.ClosedAt), "not closed"),
					ReopenedAt:   expectedDate(derefInt64(item.ReopenedAt), "never reopened"),
				}
				if item.ReopenReason != "" && gate.show("reopenReason", "reopenReason") {
					row.ReopenReason = item.ReopenReason
				}

				return row
			}), nil
		},
	})
}

type closeBlockerReader interface {
	Get(
		ctx context.Context,
		req repositories.GetFiscalPeriodByIDRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	GetCloseBlockers(
		ctx context.Context,
		req repositories.GetFiscalPeriodByIDRequest,
	) (*fiscalclose.Result, error)
}

type getFiscalCloseBlockersTool struct {
	periods closeBlockerReader
}

func newGetFiscalCloseBlockersTool(periods closeBlockerReader) serviceports.AgentQueryTool {
	return &getFiscalCloseBlockersTool{periods: periods}
}

func (t *getFiscalCloseBlockersTool) Name() string { return "get_fiscal_close_blockers" }

func (t *getFiscalCloseBlockersTool) Description() string {
	return "Say whether a fiscal period can be closed and list every blocker in the way. " +
		"Blockers include unposted entries, earlier periods still open, subledger " +
		"differences and accounting controls; nothing changes. Take the period's id from " +
		"list_fiscal_periods."
}

func (t *getFiscalCloseBlockersTool) ParamSchema() map[string]any {
	return idSchema(paramFiscalPeriodID, "The fiscal period's id, from list_fiscal_periods or "+
		onThePage)
}

func (t *getFiscalCloseBlockersTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFiscalPeriod})
}

type closeBlockerRow struct {
	Category string `json:"category"`
	Field    string `json:"field,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
}

type closeBlockersView struct {
	FiscalPeriodID  string            `json:"fiscalPeriodId"`
	Name            string            `json:"name,omitempty"`
	Status          string            `json:"status,omitempty"`
	CanClose        bool              `json:"canClose"`
	BlockerCount    int               `json:"blockerCount"`
	Blockers        []closeBlockerRow `json:"blockers"`
	BlockersOmitted int               `json:"blockersOmitted,omitempty"`
}

func (t *getFiscalCloseBlockersTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, paramFiscalPeriodID)
	if err != nil {
		return nil, err
	}

	req := repositories.GetFiscalPeriodByIDRequest{ID: id, TenantInfo: tenantOf(params)}
	period, err := t.periods.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	result, err := t.periods.GetCloseBlockers(ctx, req)
	if err != nil {
		return nil, err
	}

	view := closeBlockersView{
		FiscalPeriodID: id.String(),
		Name:           period.Name,
		Status:         string(period.Status),
		CanClose:       result.CanClose,
		BlockerCount:   len(result.Blockers),
		Blockers:       make([]closeBlockerRow, 0, min(len(result.Blockers), maxCloseBlockers)),
	}
	for _, blocker := range result.Blockers {
		if blocker == nil {
			continue
		}
		if len(view.Blockers) == maxCloseBlockers {
			view.BlockersOmitted++
			continue
		}
		view.Blockers = append(view.Blockers, closeBlockerRow{
			Category: blocker.Category,
			Field:    blocker.Field,
			Code:     string(blocker.Code),
			Message:  blocker.Message,
		})
	}

	return view, nil
}
