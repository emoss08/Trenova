package accountsreceivableservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetAgingSummaryAggregatesTotals(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		ListARAging(mock.Anything, mock.Anything).
		Return([]*repositories.ARCustomerAgingRow{
			{
				CustomerID: pulid.MustNew("cus_"),
				Buckets: repositories.ARAgingBucketTotals{
					CurrentMinor:   100,
					Days1To30Minor: 200,
					TotalOpenMinor: 300,
				},
			},
			{
				CustomerID: pulid.MustNew("cus_"),
				Buckets: repositories.ARAgingBucketTotals{
					Days31To60Minor: 400,
					TotalOpenMinor:  400,
				},
			},
		}, nil).
		Once()
	svc := &Service{repo: repo}

	summary, err := svc.GetAgingSummary(t.Context(), pagination.TenantInfo{}, 123)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, int64(100), summary.Totals.CurrentMinor)
	assert.Equal(t, int64(200), summary.Totals.Days1To30Minor)
	assert.Equal(t, int64(400), summary.Totals.Days31To60Minor)
	assert.Equal(t, int64(700), summary.Totals.TotalOpenMinor)
}

func TestListOpenItemsDefaultsAsOfDate(t *testing.T) {
	t.Parallel()

	expected := []*repositories.AROpenItem{{InvoiceID: pulid.MustNew("inv_")}}
	repo := mocks.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		ListOpenItems(mock.Anything, mock.Anything).
		Return(expected, nil).
		Once()
	svc := &Service{repo: repo}

	items, err := svc.ListOpenItems(t.Context(), pagination.TenantInfo{}, pulid.ID(""), 0)

	require.NoError(t, err)
	require.Equal(t, expected, items)
}

func TestGetCustomerStatementBuildsBalanceForwardAndRunningBalance(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := mocks.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		GetCustomerName(mock.Anything, mock.Anything).
		Return("Acme", nil).
		Once()
	repo.EXPECT().
		ListCustomerLedger(mock.Anything, mock.Anything).
		Return([]*repositories.ARLedgerEntry{
			{
				CustomerID:      customerID,
				TransactionDate: 10,
				EventType:       "InvoicePosted",
				DocumentNumber:  "INV-1",
				AmountMinor:     10000,
			},
			{
				CustomerID:      customerID,
				TransactionDate: 20,
				EventType:       "CustomerPaymentPosted",
				DocumentNumber:  "PAY-1",
				AmountMinor:     -4000,
			},
			{
				CustomerID:      customerID,
				TransactionDate: 30,
				EventType:       "InvoicePosted",
				DocumentNumber:  "INV-2",
				AmountMinor:     3000,
			},
		}, nil).
		Once()
	repo.EXPECT().
		ListOpenItems(mock.Anything, mock.Anything).
		Return([]*repositories.AROpenItem{
			{InvoiceID: pulid.MustNew("inv_"), CustomerID: customerID, OpenAmountMinor: 9000},
		}, nil).
		Once()
	repo.EXPECT().
		GetCustomerAging(mock.Anything, mock.Anything).
		Return(&repositories.ARCustomerAgingRow{
			CustomerID:   customerID,
			CustomerName: "Acme",
			Buckets:      repositories.ARAgingBucketTotals{TotalOpenMinor: 9000},
		}, nil).
		Once()
	svc := &Service{repo: repo}

	statement, err := svc.GetCustomerStatement(
		t.Context(),
		pagination.TenantInfo{},
		customerID,
		15,
		40,
	)

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
