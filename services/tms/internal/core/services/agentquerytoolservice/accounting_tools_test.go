package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceivables struct {
	aging     *serviceports.ARAgingSummary
	openItems []*repositories.AROpenItem
	statement *serviceports.ARCustomerStatement
	worklist  []*repositories.ARCollectionsWorklistItem

	openCustomer pulid.ID
	worklistReq  repositories.ListARCollectionsWorklistRequest
}

func (f *fakeReceivables) GetAgingSummary(
	context.Context,
	pagination.TenantInfo,
	int64,
) (*serviceports.ARAgingSummary, error) {
	return f.aging, nil
}

func (f *fakeReceivables) ListOpenItems(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	_ int64,
) ([]*repositories.AROpenItem, error) {
	f.openCustomer = customerID

	return f.openItems, nil
}

func (f *fakeReceivables) GetCustomerStatement(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	int64, int64,
) (*serviceports.ARCustomerStatement, error) {
	return f.statement, nil
}

func (f *fakeReceivables) GetCollectionsWorklist(
	_ context.Context,
	req repositories.ListARCollectionsWorklistRequest,
) ([]*repositories.ARCollectionsWorklistItem, error) {
	f.worklistReq = req

	return f.worklist, nil
}

func agingRow(name string, buckets repositories.ARAgingBucketTotals) *repositories.ARCustomerAgingRow {
	return &repositories.ARCustomerAgingRow{
		CustomerID:   pulid.MustNew("cus_"),
		CustomerName: name,
		Buckets:      buckets,
	}
}

func agingSummary() *serviceports.ARAgingSummary {
	return &serviceports.ARAgingSummary{
		AsOfDate: 1_700_000_000,
		Totals:   repositories.ARAgingBucketTotals{TotalOpenMinor: 160000, DaysOver90Minor: 50000},
		Rows: []*repositories.ARCustomerAgingRow{
			agingRow("Current Co", repositories.ARAgingBucketTotals{
				CurrentMinor: 100000, TotalOpenMinor: 100000,
			}),
			agingRow("Slow Payer", repositories.ARAgingBucketTotals{
				DaysOver90Minor: 50000, TotalOpenMinor: 50000,
			}),
			agingRow("Paid Up", repositories.ARAgingBucketTotals{}),
			agingRow("Month Late", repositories.ARAgingBucketTotals{
				Days31To60Minor: 10000, TotalOpenMinor: 10000,
			}),
		},
	}
}

func TestGetARAging_WithholdsAmountsButStillRanksByAge(t *testing.T) {
	t.Parallel()

	tool := newGetARAgingTool(&fakeReceivables{aging: agingSummary()}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)

	outcome, ok := result.(arAgingOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]arAgingRow)
	require.True(t, ok)
	require.Len(t, rows, 3, "a customer with nothing open is not a receivable")
	assert.Equal(t, "Slow Payer", rows[0].Customer)
	assert.Equal(t, bucketOver90, rows[0].OldestBucket)
	assert.Equal(t, "Month Late", rows[1].Customer)
	assert.Empty(t, rows[0].TotalOpen)
	assert.Nil(t, outcome.Totals)
	assert.Equal(t, []string{"amounts"}, outcome.Withheld)
}

func TestGetARAging_ShowsAmountsAtRestricted(t *testing.T) {
	t.Parallel()

	tool := newGetARAgingTool(&fakeReceivables{aging: agingSummary()}, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"overdueOnly": true}, permission.SensitivityRestricted))
	require.NoError(t, err)

	outcome := result.(arAgingOutcome)
	rows := outcome.Items.([]arAgingRow)
	require.Len(t, rows, 2, "overdueOnly drops the customer whose balance is current")
	assert.Equal(t, "500.00", rows[0].TotalOpen)
	require.NotNil(t, outcome.Totals)
	assert.Equal(t, "1600.00", outcome.Totals.TotalOpen)
	assert.Empty(t, outcome.Withheld)
}

func TestListAROpenItems_FiltersAndPages(t *testing.T) {
	t.Parallel()

	customer := pulid.MustNew("cus_")
	fake := &fakeReceivables{openItems: []*repositories.AROpenItem{
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "A", DaysPastDue: 5},
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "B", DaysPastDue: 45,
			DisputeStatus: "Disputed", OpenAmountMinor: 1234},
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "C", DaysPastDue: 70},
	}}
	tool := newListAROpenItemsTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"customerId":     customer.String(),
		"minDaysPastDue": float64(30),
		"limit":          float64(1),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	assert.Equal(t, customer, fake.openCustomer, "the customer narrows the query itself")
	outcome := result.(*gatedOutcome)
	rows := outcome.Items.([]arOpenItemRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "C", rows[0].InvoiceNumber, "most overdue first")
	assert.Equal(t, 2, outcome.Count)
	assert.True(t, outcome.HasMore)

	result, err = tool.Query(t.Context(),
		agentParams(map[string]any{"disputedOnly": true}, permission.SensitivityRestricted))
	require.NoError(t, err)
	rows = result.(*gatedOutcome).Items.([]arOpenItemRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "12.34", rows[0].Open)
}

func TestGetCustomerStatement_KeepsTheLatestTransactions(t *testing.T) {
	t.Parallel()

	transactions := make([]*serviceports.ARStatementTransaction, 0, 60)
	for idx := range 60 {
		transactions = append(transactions, &serviceports.ARStatementTransaction{
			TransactionDate: int64(1_700_000_000 + idx),
			EventType:       "Invoice",
			ChargeMinor:     100,
		})
	}
	customer := pulid.MustNew("cus_")
	tool := newGetCustomerStatementTool(&fakeReceivables{statement: &serviceports.ARCustomerStatement{
		CustomerID:   customer,
		CustomerName: "Acme",
		Transactions: transactions,
	}}, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"customerId": customer.String()}, ""))
	require.NoError(t, err)

	view := result.(customerStatementView)
	assert.Len(t, view.Transactions, maxStatementRows)
	assert.Equal(t, 10, view.TransactionsOmitted)
	assert.Empty(t, view.Transactions[0].Charge)
	assert.Empty(t, view.EndingBalance)
	assert.Contains(t, view.Withheld, "amounts")

	_, err = tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.Error(t, err, "a statement is for one customer")
}

func TestListCollectionsWorklist_NarrowsToASeverity(t *testing.T) {
	t.Parallel()

	fake := &fakeReceivables{worklist: []*repositories.ARCollectionsWorklistItem{
		{InvoiceID: pulid.MustNew("inv_"), Severity: "Critical", DaysPastDue: 40},
		{InvoiceID: pulid.MustNew("inv_"), Severity: "Watch", DaysPastDue: 2},
	}}
	tool := newListCollectionsWorklistTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"severity": "Critical",
		"limit":    float64(5),
	}, ""))
	require.NoError(t, err)

	outcome := result.(*gatedOutcome)
	rows := outcome.Items.([]worklistRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Critical", rows[0].Severity)
	assert.Equal(t, maxAccountingRows, fake.worklistReq.Limit,
		"a severity is filtered over the whole worklist, not the first page of it")
}

type fakeJournalEntries struct {
	entries  []*journalentry.JournalEntry
	captured *repositories.ListJournalEntriesRequest
}

func (f *fakeJournalEntries) List(
	_ context.Context,
	req *repositories.ListJournalEntriesRequest,
) (*pagination.ListResult[*journalentry.JournalEntry], error) {
	f.captured = req

	return &pagination.ListResult[*journalentry.JournalEntry]{Items: f.entries}, nil
}

func (f *fakeJournalEntries) GetByID(
	_ context.Context,
	req repositories.GetJournalEntryByIDRequest,
) (*journalentry.JournalEntry, error) {
	for _, entry := range f.entries {
		if entry.ID == req.ID {
			return entry, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Journal entry not found")
}

type fakeGLAccounts struct {
	accounts []*glaccount.GLAccount
}

func (f *fakeGLAccounts) GetByIDs(
	context.Context,
	repositories.GetGLAccountsByIDsRequest,
) ([]*glaccount.GLAccount, error) {
	return f.accounts, nil
}

func TestListJournalEntries_PassesTheNarrowingToTheRepository(t *testing.T) {
	t.Parallel()

	fake := &fakeJournalEntries{}
	tool := &listJournalEntriesTool{entries: fake, access: newFieldAccess(&fakePermissions{})}
	period := pulid.MustNew("fp_")

	_, err := tool.Query(t.Context(), agentParams(map[string]any{
		"status":         "Posted",
		"fiscalPeriodId": period.String(),
		"fromDate":       "2026-03-01",
		"toDate":         "2026-03-31",
	}, ""))
	require.NoError(t, err)

	require.NotNil(t, fake.captured)
	assert.Equal(t, "Posted", fake.captured.Status)
	assert.Equal(t, period, fake.captured.FiscalPeriodID)
	assert.Positive(t, fake.captured.AccountingDateStart)
	assert.Greater(t, fake.captured.AccountingDateEnd, fake.captured.AccountingDateStart)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"status": "Booked"}, ""))
	require.Error(t, err)
}

func TestGetJournalEntry_NamesAccountsAndWithholdsAmounts(t *testing.T) {
	t.Parallel()

	account := &glaccount.GLAccount{
		ID:          pulid.MustNew("gla_"),
		AccountCode: "5100",
		Name:        "Fuel expense",
	}
	entry := &journalentry.JournalEntry{
		ID:             pulid.MustNew("je_"),
		EntryNumber:    "JE-7",
		TotalDebit:     5000,
		ReversalReason: "Posted to the wrong period",
		Lines: []*journalentry.JournalEntryLine{
			{LineNumber: 1, GLAccountID: account.ID, DebitAmount: 5000},
		},
	}
	tool := &getJournalEntryTool{
		entries:  &fakeJournalEntries{entries: []*journalentry.JournalEntry{entry}},
		accounts: &fakeGLAccounts{accounts: []*glaccount.GLAccount{account}},
		access:   newFieldAccess(&fakePermissions{}),
	}

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"journalEntryId": entry.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(journalEntryView)
	require.Len(t, view.Lines, 1)
	assert.Equal(t, "5100", view.Lines[0].AccountCode)
	assert.Equal(t, "Fuel expense", view.Lines[0].AccountName)
	assert.Empty(t, view.Lines[0].Debit)
	assert.Empty(t, view.TotalDebit)
	assert.Empty(t, view.ReversalReason)
	assert.Subset(t, view.Withheld, []string{"totalDebit", "lines.debit", "reversalReason"})
}

type fakeCloseBlockers struct {
	period *fiscalperiod.FiscalPeriod
	result *fiscalclose.Result
}

func (f *fakeCloseBlockers) Get(
	context.Context,
	repositories.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return f.period, nil
}

func (f *fakeCloseBlockers) GetCloseBlockers(
	context.Context,
	repositories.GetFiscalPeriodByIDRequest,
) (*fiscalclose.Result, error) {
	return f.result, nil
}

func TestGetFiscalCloseBlockers_SaysWhatStopsTheClose(t *testing.T) {
	t.Parallel()

	period := &fiscalperiod.FiscalPeriod{
		ID:     pulid.MustNew("fp_"),
		Name:   "March 2026",
		Status: fiscalperiod.StatusOpen,
	}
	tool := newGetFiscalCloseBlockersTool(&fakeCloseBlockers{
		period: period,
		result: &fiscalclose.Result{Blockers: []*fiscalclose.Blocker{
			{Category: "period", Message: "February is still open"},
		}},
	})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"fiscalPeriodId": period.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(closeBlockersView)
	assert.False(t, view.CanClose)
	assert.Equal(t, "March 2026", view.Name)
	require.Len(t, view.Blockers, 1)
	assert.Equal(t, "February is still open", view.Blockers[0].Message)
}
