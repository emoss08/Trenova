package accountsreceivableservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAgingSummaryAggregatesTotals(t *testing.T) {
	t.Parallel()

	repo := fakeARRepo{rows: []*accountsreceivable.CustomerAgingRow{{CustomerID: pulid.MustNew("cus_"), Buckets: accountsreceivable.AgingBucketTotals{CurrentMinor: 100, Days1To30Minor: 200, TotalOpenMinor: 300}}, {CustomerID: pulid.MustNew("cus_"), Buckets: accountsreceivable.AgingBucketTotals{Days31To60Minor: 400, TotalOpenMinor: 400}}}}
	svc := &Service{repo: repo}

	summary, err := svc.GetAgingSummary(t.Context(), pagination.TenantInfo{}, 123)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(100), summary.Totals.CurrentMinor)
	assert.Equal(t, int64(200), summary.Totals.Days1To30Minor)
	assert.Equal(t, int64(400), summary.Totals.Days31To60Minor)
	assert.Equal(t, int64(700), summary.Totals.TotalOpenMinor)
}

type fakeARRepo struct {
	ledger []*accountsreceivable.LedgerEntry
	rows   []*accountsreceivable.CustomerAgingRow
	items  []*accountsreceivable.OpenItem
	name   string
	aging  *accountsreceivable.CustomerAgingRow
}

func (f fakeARRepo) ListCustomerLedger(context.Context, repositories.ListCustomerLedgerRequest) ([]*accountsreceivable.LedgerEntry, error) {
	return f.ledger, nil
}

func (f fakeARRepo) ListARAging(context.Context, repositories.ListARAgingRequest) ([]*accountsreceivable.CustomerAgingRow, error) {
	return f.rows, nil
}

func (f fakeARRepo) ListOpenItems(context.Context, repositories.ListAROpenItemsRequest) ([]*accountsreceivable.OpenItem, error) {
	return f.items, nil
}

func (f fakeARRepo) GetCustomerName(context.Context, repositories.GetARCustomerNameRequest) (string, error) {
	return f.name, nil
}

func (f fakeARRepo) GetCustomerAging(context.Context, repositories.GetARCustomerAgingRequest) (*accountsreceivable.CustomerAgingRow, error) {
	return f.aging, nil
}

func TestListOpenItemsDefaultsAsOfDate(t *testing.T) {
	t.Parallel()

	expected := []*accountsreceivable.OpenItem{{InvoiceID: pulid.MustNew("inv_")}}
	repo := fakeARRepo{items: expected}
	svc := &Service{repo: repo}

	items, err := svc.ListOpenItems(t.Context(), pagination.TenantInfo{}, pulid.ID(""), 0)

	require.NoError(t, err)
	require.Equal(t, expected, items)
}

func TestGetCustomerStatementBuildsBalanceForwardAndRunningBalance(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := fakeARRepo{
		name: "Acme",
		ledger: []*accountsreceivable.LedgerEntry{
			{CustomerID: customerID, TransactionDate: 10, EventType: "InvoicePosted", DocumentNumber: "INV-1", AmountMinor: 10000},
			{CustomerID: customerID, TransactionDate: 20, EventType: "CustomerPaymentPosted", DocumentNumber: "PAY-1", AmountMinor: -4000},
			{CustomerID: customerID, TransactionDate: 30, EventType: "InvoicePosted", DocumentNumber: "INV-2", AmountMinor: 3000},
		},
		items: []*accountsreceivable.OpenItem{{InvoiceID: pulid.MustNew("inv_"), CustomerID: customerID, OpenAmountMinor: 9000}},
		aging: &accountsreceivable.CustomerAgingRow{CustomerID: customerID, CustomerName: "Acme", Buckets: accountsreceivable.AgingBucketTotals{TotalOpenMinor: 9000}},
	}
	svc := &Service{repo: repo}

	statement, err := svc.GetCustomerStatement(t.Context(), pagination.TenantInfo{}, customerID, 15, 40)

	require.NoError(t, err)
	require.NotNil(t, statement)
	assert.Equal(t, "Acme", statement.CustomerName)
	assert.Equal(t, int64(10000), statement.OpeningBalanceMinor)
	assert.Equal(t, int64(3000), statement.TotalChargesMinor)
	assert.Equal(t, int64(4000), statement.TotalPaymentsMinor)
	assert.Equal(t, int64(9000), statement.EndingBalanceMinor)
	require.Len(t, statement.Transactions, 2)
	assert.Equal(t, int64(6000), statement.Transactions[0].RunningBalanceMinor)
	assert.Equal(t, int64(9000), statement.Transactions[1].RunningBalanceMinor)
	assert.Equal(t, int64(9000), statement.Aging.TotalOpenMinor)
}
