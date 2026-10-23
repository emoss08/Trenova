package agentquerytoolservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	defaultAccountingRows = 25
	maxAccountingRows     = 100
	maxStatementRows      = 50
	maxCloseBlockers      = 50

	bucketCurrent = "current"
	bucket1To30   = "1-30"
	bucket31To60  = "31-60"
	bucket61To90  = "61-90"
	bucketOver90  = "over 90"
)

func accountingToolProviders() []any {
	return []any{
		provideGetARAgingTool,
		provideListAROpenItemsTool,
		provideGetCustomerStatementTool,
		provideListCollectionsWorklistTool,
		newListGLAccountsTool,
		provideGetFiscalCloseBlockersTool,
	}
}

type receivablesReader interface {
	GetAgingSummary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		asOfDate int64,
	) (*customerledgerports.ARAgingSummary, error)
	ListOpenItems(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		asOfDate int64,
	) ([]*customerledgerports.AROpenItem, error)
	GetCustomerStatement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
		startDate, asOfDate int64,
	) (*customerledgerports.ARCustomerStatement, error)
	GetCollectionsWorklist(
		ctx context.Context,
		req customerledgerports.ListARCollectionsWorklistRequest,
	) ([]*customerledgerports.ARCollectionsWorklistItem, error)
}

func provideGetARAgingTool(
	receivables *accountsreceivableservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetARAgingTool(receivables, permissions)
}

func provideListAROpenItemsTool(
	receivables *accountsreceivableservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListAROpenItemsTool(receivables, permissions)
}

func provideGetCustomerStatementTool(
	receivables *accountsreceivableservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetCustomerStatementTool(receivables, permissions)
}

func provideListCollectionsWorklistTool(
	receivables *accountsreceivableservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListCollectionsWorklistTool(receivables, permissions)
}

func provideGetFiscalCloseBlockersTool(
	periods *fiscalperiodservice.Service,
) agentports.AgentQueryTool {
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

func arBucketsFrom(totals customerledgerports.ARAgingBucketTotals) *arBuckets {
	return &arBuckets{
		Current:    minorText(totals.CurrentMinor),
		Days1To30:  minorText(totals.Days1To30Minor),
		Days31To60: minorText(totals.Days31To60Minor),
		Days61To90: minorText(totals.Days61To90Minor),
		Over90:     minorText(totals.DaysOver90Minor),
		TotalOpen:  minorText(totals.TotalOpenMinor),
	}
}

func oldestBucket(totals customerledgerports.ARAgingBucketTotals) (string, int) {
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getARAgingTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *getARAgingTool) Name() string { return "receivable_get_aging" }

func (t *getARAgingTool) Description() string {
	return "Summarize accounts receivable aging: which customers owe money and how overdue " +
		"it is, in current, 1-30, 31-60, 61-90 and over-90-day buckets. Each row names the " +
		"oldest bucket holding a balance. Use receivable_list_open_items for the invoices behind a " +
		"customer's balance and receivable_get_statement for their history. Amounts are " +
		"named in withheldByAccess when your data access does not reach them."
}

func (t *getARAgingTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramAsOf: agenttoolschema.Date("The day to age as of. Defaults to today."),
		paramCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"Only this customer, by id from customer_list or a row of this tool."),
		"overdueOnly": boolParam("Only customers with a balance past due."),
	}, defaultAccountingRows, maxAccountingRows))
}

func (t *getARAgingTool) Policy() agentports.ToolPolicy {
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
	params *agentports.QueryToolParams,
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

func agingRows(entries []*customerledgerports.ARCustomerAgingRow, filter agingFilter) []arAgingRow {
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listAROpenItemsTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *listAROpenItemsTool) Name() string { return "receivable_list_open_items" }

func (t *listAROpenItemsTool) Description() string {
	return "List the open receivables, the unpaid or partly paid invoices customers still " +
		"owe, most overdue first. Each row has the due date, days past due, dispute and " +
		"short-pay state, and the open balance. Narrow to one customer, a minimum number " +
		"of days past due, or disputed invoices. invoice_get expands one."
}

func (t *listAROpenItemsTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"Only this customer's invoices, by id from customer_list or receivable_get_aging."),
		"minDaysPastDue": intParam("Only invoices at least this many days past due."),
		"disputedOnly":   boolParam("Only invoices the customer disputes."),
		paramAsOf:        agenttoolschema.Date("The day to age as of. Defaults to today."),
	}, defaultAccountingRows, maxAccountingRows))
}

func (t *listAROpenItemsTool) Policy() agentports.ToolPolicy {
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

func arOpenItemRowFrom(item *customerledgerports.AROpenItem, showAmounts bool) arOpenItemRow {
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
	params *agentports.QueryToolParams,
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

	kept := make([]*customerledgerports.AROpenItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.DaysPastDue < minDays {
			continue
		}
		if disputedOnly && !strings.EqualFold(item.DisputeStatus, "Disputed") {
			continue
		}
		kept = append(kept, item)
	}
	slices.SortStableFunc(kept, func(a, b *customerledgerports.AROpenItem) int {
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getCustomerStatementTool{receivables: receivables, access: newFieldAccess(permissions)}
}

func (t *getCustomerStatementTool) Name() string { return "receivable_get_statement" }

func (t *getCustomerStatementTool) Description() string {
	return "Build one customer's account statement: invoices, payments and credits over a " +
		"period with the running balance, what is still open, and their aging. It answers " +
		"what a customer was billed and paid and what they owe now. Take the customer's " +
		"id from customer_list or receivable_get_aging."
}

func (t *getCustomerStatementTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"The customer's id, from customer_list, receivable_get_aging or "+onThePage),
		paramStartDate: agenttoolschema.Date(
			"The first day of the statement period. Omit for the " +
				"customer's whole history.",
		),
		paramAsOf: agenttoolschema.Date("The statement date. Defaults to today."),
	}, paramCustomerID)
}

func (t *getCustomerStatementTool) Policy() agentports.ToolPolicy {
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
	params *agentports.QueryToolParams,
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
	transactions []*customerledgerports.ARStatementTransaction,
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
	items []*customerledgerports.AROpenItem,
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listCollectionsWorklistTool{
		receivables: receivables,
		access:      newFieldAccess(permissions),
	}
}

func (t *listCollectionsWorklistTool) Name() string { return "receivable_list_collection_worklist" }

func (t *listCollectionsWorklistTool) Description() string {
	return "List the collections worklist: overdue and disputed invoices ranked for a " +
		"collector to chase, each marked Critical, Warning or Watch. Rows carry the " +
		"customer, days past due, the open dispute's reason and the open amount. Use " +
		"receivable_get_statement before calling a customer about their balance."
}

func (t *listCollectionsWorklistTool) SearchTerms() []string {
	return []string{"collections worklist"}
}

func (t *listCollectionsWorklistTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramAsOf:     agenttoolschema.Date("The day to rank as of. Defaults to today."),
		paramSeverity: enumParam("Only rows of this severity.", accountingDriftSeverities),
		paramLimit:    limitSchema(defaultAccountingRows, maxAccountingRows),
	})
}

func (t *listCollectionsWorklistTool) Policy() agentports.ToolPolicy {
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
	params *agentports.QueryToolParams,
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
		customerledgerports.ListARCollectionsWorklistRequest{
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
			InvoiceID:     item.InvoiceID.String(),
			InvoiceNumber: item.InvoiceNumber,
			CustomerID:    item.CustomerID.String(),
			Customer:      item.CustomerName,
			Severity:      item.Severity,
			DueDate:       recordedDate(item.DueDate),
			DaysPastDue:   max(item.DaysPastDue, 0),
			Disputed:      item.IsDisputed,
			HasShortPay:   item.HasShortPay,
			DisputeReason: item.OpenDisputeReasonCode,
			DisputeOpenedAt: expectedDate(
				typeutils.ValueOrZero(item.DisputeOpenedAt),
				"no open dispute",
			),
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
	repo glaccountports.GLAccountRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "ledger_list_accounts",
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
			_ pagination.CursorInfo,
			gate *fieldGate,
		) (listPage, error) {
			result, err := repo.List(ctx, &glaccountports.ListGLAccountsRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			showCurrent := gate.show("currentBalance", "currentBalance")
			showDebit := gate.show("debitBalance", "debitBalance")
			showCredit := gate.show("creditBalance", "creditBalance")

			return offsetListPage(result.Items, func(item *glaccount.GLAccount) any {
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
	repo fiscalperiodports.FiscalPeriodRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "ledger_list_fiscal_periods",
		entityPlural: "fiscal periods",
		searchTerms: []string{
			"open periods", "period end dates", "current period", "fiscal calendar",
		},
		summary: "List fiscal periods, the months, quarters or adjusting periods the books " +
			"close by, with their dates and whether each is open, locked or closed. " +
			"ledger_get_fiscal_close_blockers says what stops one from closing.",
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
			_ pagination.CursorInfo,
			gate *fieldGate,
		) (listPage, error) {
			result, err := repo.List(ctx, &fiscalperiodports.ListFiscalPeriodsRequest{Filter: opts})
			if err != nil {
				return listPage{}, err
			}

			return offsetListPage(result.Items, func(item *fiscalperiod.FiscalPeriod) any {
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
					LockedAt:     expectedDate(typeutils.ValueOrZero(item.LockedAt), "not locked"),
					ClosedAt:     expectedDate(typeutils.ValueOrZero(item.ClosedAt), "not closed"),
					ReopenedAt: expectedDate(
						typeutils.ValueOrZero(item.ReopenedAt),
						"never reopened",
					),
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
		req fiscalperiodports.GetFiscalPeriodByIDRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	GetCloseBlockers(
		ctx context.Context,
		req fiscalperiodports.GetFiscalPeriodByIDRequest,
	) (*fiscalclose.Result, error)
}

type getFiscalCloseBlockersTool struct {
	periods closeBlockerReader
}

func newGetFiscalCloseBlockersTool(periods closeBlockerReader) agentports.AgentQueryTool {
	return &getFiscalCloseBlockersTool{periods: periods}
}

func (t *getFiscalCloseBlockersTool) Name() string { return "ledger_get_fiscal_close_blockers" }

func (t *getFiscalCloseBlockersTool) SearchTerms() []string {
	return []string{
		"blocking", "blocked", "closing", "period", "ready", "close the books", "month end close",
	}
}

func (t *getFiscalCloseBlockersTool) Description() string {
	return "Say whether a fiscal period can be closed and list every blocker in the way. " +
		"Blockers include unposted entries, earlier periods still open, subledger " +
		"differences and accounting controls; nothing changes. Take the period's id from " +
		"ledger_list_fiscal_periods."
}

func (t *getFiscalCloseBlockersTool) ParamSchema() map[string]any {
	return idSchema(
		paramFiscalPeriodID,
		agenttoolschema.RecordIDText(permission.ResourceFiscalPeriod,
			"The fiscal period's id, from ledger_list_fiscal_periods or "+onThePage),
	)
}

func (t *getFiscalCloseBlockersTool) Policy() agentports.ToolPolicy {
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
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, paramFiscalPeriodID)
	if err != nil {
		return nil, err
	}

	req := fiscalperiodports.GetFiscalPeriodByIDRequest{ID: id, TenantInfo: tenantOf(params)}
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
