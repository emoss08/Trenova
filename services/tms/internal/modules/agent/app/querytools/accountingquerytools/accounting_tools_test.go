package accountingquerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReceivables struct {
	aging     *customerledgerports.ARAgingSummary
	openItems []*customerledgerports.AROpenItem
	statement *customerledgerports.ARCustomerStatement
	worklist  []*customerledgerports.ARCollectionsWorklistItem

	openCustomer pulid.ID
	worklistReq  customerledgerports.ListARCollectionsWorklistRequest
}

func (f *fakeReceivables) GetAgingSummary(
	context.Context,
	pagination.TenantInfo,
	int64,
) (*customerledgerports.ARAgingSummary, error) {
	return f.aging, nil
}

func (f *fakeReceivables) ListOpenItems(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	_ int64,
) ([]*customerledgerports.AROpenItem, error) {
	f.openCustomer = customerID

	return f.openItems, nil
}

func (f *fakeReceivables) GetCustomerStatement(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	int64, int64,
) (*customerledgerports.ARCustomerStatement, error) {
	return f.statement, nil
}

func (f *fakeReceivables) GetCollectionsWorklist(
	_ context.Context,
	req customerledgerports.ListARCollectionsWorklistRequest,
) ([]*customerledgerports.ARCollectionsWorklistItem, error) {
	f.worklistReq = req

	return f.worklist, nil
}

func agingRow(
	name string,
	buckets customerledgerports.ARAgingBucketTotals,
) *customerledgerports.ARCustomerAgingRow {
	return &customerledgerports.ARCustomerAgingRow{
		CustomerID:   pulid.MustNew("cus_"),
		CustomerName: name,
		Buckets:      buckets,
	}
}

func agingSummary() *customerledgerports.ARAgingSummary {
	return &customerledgerports.ARAgingSummary{
		AsOfDate: 1_700_000_000,
		Totals: customerledgerports.ARAgingBucketTotals{
			TotalOpenMinor:  160000,
			DaysOver90Minor: 50000,
		},
		Rows: []*customerledgerports.ARCustomerAgingRow{
			agingRow("Current Co", customerledgerports.ARAgingBucketTotals{
				CurrentMinor: 100000, TotalOpenMinor: 100000,
			}),
			agingRow("Slow Payer", customerledgerports.ARAgingBucketTotals{
				DaysOver90Minor: 50000, TotalOpenMinor: 50000,
			}),
			agingRow("Paid Up", customerledgerports.ARAgingBucketTotals{}),
			agingRow("Month Late", customerledgerports.ARAgingBucketTotals{
				Days31To60Minor: 10000, TotalOpenMinor: 10000,
			}),
		},
	}
}

func TestGetARAging_WithholdsAmountsButStillRanksByAge(t *testing.T) {
	t.Parallel()

	tool := newGetARAgingTool(
		&fakeReceivables{aging: agingSummary()},
		&querytooltest.FakePermissions{},
	)

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{}, ""))
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

	tool := newGetARAgingTool(
		&fakeReceivables{aging: agingSummary()},
		&querytooltest.FakePermissions{},
	)

	result, err := tool.Query(
		t.Context(),
		querytooltest.AgentParams(
			map[string]any{"overdueOnly": true},
			permission.SensitivityRestricted,
		),
	)
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
	fake := &fakeReceivables{openItems: []*customerledgerports.AROpenItem{
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "A", DaysPastDue: 5},
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "B", DaysPastDue: 45,
			DisputeStatus: "Disputed", OpenAmountMinor: 1234},
		{InvoiceID: pulid.MustNew("inv_"), InvoiceNumber: "C", DaysPastDue: 70},
	}}
	tool := newListAROpenItemsTool(fake, &querytooltest.FakePermissions{})

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"customerId":     customer.String(),
		"minDaysPastDue": float64(30),
		"limit":          float64(1),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	assert.Equal(t, customer, fake.openCustomer, "the customer narrows the query itself")
	outcome := result.(*fieldgate.Outcome)
	rows := outcome.Items.([]arOpenItemRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "C", rows[0].InvoiceNumber, "most overdue first")
	assert.Equal(t, 2, outcome.Count)
	assert.True(t, outcome.HasMore)

	result, err = tool.Query(
		t.Context(),
		querytooltest.AgentParams(
			map[string]any{"disputedOnly": true},
			permission.SensitivityRestricted,
		),
	)
	require.NoError(t, err)
	rows = result.(*fieldgate.Outcome).Items.([]arOpenItemRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "12.34", rows[0].Open)
}

func TestGetCustomerStatement_KeepsTheLatestTransactions(t *testing.T) {
	t.Parallel()

	transactions := make([]*customerledgerports.ARStatementTransaction, 0, 60)
	for idx := range 60 {
		transactions = append(transactions, &customerledgerports.ARStatementTransaction{
			TransactionDate: int64(1_700_000_000 + idx),
			EventType:       "Invoice",
			ChargeMinor:     100,
		})
	}
	customer := pulid.MustNew("cus_")
	tool := newGetCustomerStatementTool(
		&fakeReceivables{statement: &customerledgerports.ARCustomerStatement{
			CustomerID:   customer,
			CustomerName: "Acme",
			Transactions: transactions,
		}},
		&querytooltest.FakePermissions{},
	)

	result, err := tool.Query(t.Context(),
		querytooltest.AgentParams(map[string]any{"customerId": customer.String()}, ""))
	require.NoError(t, err)

	view := result.(customerStatementView)
	assert.Len(t, view.Transactions, maxStatementRows)
	assert.Equal(t, 10, view.TransactionsOmitted)
	assert.Empty(t, view.Transactions[0].Charge)
	assert.Empty(t, view.EndingBalance)
	assert.Contains(t, view.Withheld, "amounts")

	_, err = tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{}, ""))
	require.Error(t, err, "a statement is for one customer")
}

func TestListCollectionsWorklist_NarrowsToASeverity(t *testing.T) {
	t.Parallel()

	fake := &fakeReceivables{worklist: []*customerledgerports.ARCollectionsWorklistItem{
		{InvoiceID: pulid.MustNew("inv_"), Severity: "Critical", DaysPastDue: 40},
		{InvoiceID: pulid.MustNew("inv_"), Severity: "Watch", DaysPastDue: 2},
	}}
	tool := newListCollectionsWorklistTool(fake, &querytooltest.FakePermissions{})

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"severity": "Critical",
		"limit":    float64(5),
	}, ""))
	require.NoError(t, err)

	outcome := result.(*fieldgate.Outcome)
	rows := outcome.Items.([]worklistRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Critical", rows[0].Severity)
	assert.Equal(t, maxAccountingRows, fake.worklistReq.Limit,
		"a severity is filtered over the whole worklist, not the first page of it")
}

type fakeCloseBlockers struct {
	period *fiscalperiod.FiscalPeriod
	result *fiscalclose.Result
}

func (f *fakeCloseBlockers) Get(
	context.Context,
	fiscalperiodports.GetFiscalPeriodByIDRequest,
) (*fiscalperiod.FiscalPeriod, error) {
	return f.period, nil
}

func (f *fakeCloseBlockers) GetCloseBlockers(
	context.Context,
	fiscalperiodports.GetFiscalPeriodByIDRequest,
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
		querytooltest.AgentParams(map[string]any{"fiscalPeriodId": period.ID.String()}, ""))
	require.NoError(t, err)

	view := result.(closeBlockersView)
	assert.False(t, view.CanClose)
	assert.Equal(t, "March 2026", view.Name)
	require.Len(t, view.Blockers, 1)
	assert.Equal(t, "February is still open", view.Blockers[0].Message)
}
