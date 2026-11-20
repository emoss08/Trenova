package accountsreceivablehandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/accountsreceivablehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerledgerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupHandler(
	t *testing.T,
	repo customerledgerports.AccountsReceivableRepository,
) *accountsreceivablehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: appconfig.Config{Debug: true}}
	errorHandler := helpers.NewErrorHandler(
		helpers.ErrorHandlerParams{Logger: logger, AppConfig: &cfg.App},
	)
	pm := middleware.NewPermissionMiddleware(
		middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
			ErrorHandler:     errorHandler,
		},
	)
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Timezone: "UTC"}, nil).Maybe()
	service := accountsreceivableservice.New(
		accountsreceivableservice.Params{Logger: logger, Repo: repo, Organizations: orgs},
	)

	return accountsreceivablehandler.New(
		accountsreceivablehandler.Params{
			Service:              service,
			ErrorHandler:         errorHandler,
			PermissionMiddleware: pm,
		},
	)
}

func TestHandlerCustomerStatement(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().GetCustomerName(mock.Anything, mock.Anything).Return("Acme", nil).Once()
	repo.EXPECT().
		ListOpenItems(mock.Anything, mock.Anything).
		Return([]*customerledgerports.AROpenItem{
			{InvoiceID: pulid.MustNew("inv_"), CustomerID: customerID, OpenAmountMinor: 5000},
		}, nil).
		Once()
	repo.EXPECT().
		GetCustomerLedgerBalance(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.GetCustomerLedgerBalanceRequest) bool {
				return req.CustomerID == customerID && req.Before > 0
			},
		)).
		Return(int64(2500), nil).
		Once()
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.ListCustomerLedgerWindowRequest) bool {
				return req.CustomerID == customerID &&
					req.Through-req.From >= 28*24*60*60 &&
					req.Through-req.From <= 31*24*60*60 &&
					req.After == nil &&
					req.Limit == 5001
			},
		)).
		Return([]*customerledgerports.ARLedgerEntry{
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
				AmountMinor:     -5000,
			},
		}, nil).
		Once()
	repo.EXPECT().
		GetCustomerAging(mock.Anything, mock.Anything).
		Return(&customerledgerports.ARCustomerAgingRow{
			CustomerID:   customerID,
			CustomerName: "Acme",
			Buckets:      customerledgerports.ARAgingBucketTotals{TotalOpenMinor: 5000},
		}, nil).
		Once()
	handler := setupHandler(t, repo)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/accounts-receivable/customers/" + customerID.String() + "/statement/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp customerledgerports.ARCustomerStatement
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, customerID, resp.CustomerID)
	assert.Equal(t, "Acme", resp.CustomerName)
	assert.Equal(t, int64(2500), resp.OpeningBalanceMinor)
	assert.Equal(t, int64(7500), resp.EndingBalanceMinor)
	require.Len(t, resp.Transactions, 2)
	assert.Equal(t, int64(12500), resp.Transactions[0].RunningBalanceMinor)
	assert.Equal(t, int64(7500), resp.Transactions[1].RunningBalanceMinor)
}

func TestHandlerCustomerLedgerReadsOnePage(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	entryID := pulid.MustNew("cle_")
	repo := customerledgerportsmock.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		GetCustomerLedgerBalance(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.GetCustomerLedgerBalanceRequest) bool {
				return req.CustomerID == customerID && req.Before == 100
			},
		)).
		Return(int64(4000), nil).
		Once()
	repo.EXPECT().
		ListCustomerLedgerWindow(mock.Anything, mock.MatchedBy(
			func(req customerledgerports.ListCustomerLedgerWindowRequest) bool {
				return req.CustomerID == customerID &&
					req.From == 100 && req.Through == 200 && req.Limit == 2 && req.After == nil
			},
		)).
		Return([]*customerledgerports.ARLedgerEntry{
			{
				CustomerID:      customerID,
				TransactionDate: 110,
				AmountMinor:     -1500,
				Position: customerledgerports.ARLedgerPosition{
					TransactionDate: 110, LineNumber: 1, ID: entryID,
				},
			},
			{CustomerID: customerID, TransactionDate: 120, AmountMinor: 900},
		}, nil).
		Once()
	handler := setupHandler(t, repo)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/accounts-receivable/customers/" + customerID.String() + "/ledger/").
		WithQuery(map[string]string{"startDate": "100", "endDate": "200", "first": "1"}).
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp customerledgerports.ARLedgerPage
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, int64(4000), resp.OpeningBalanceMinor)
	assert.True(t, resp.HasNextPage)
	assert.NotEmpty(t, resp.EndCursor)
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, int64(2500), resp.Entries[0].RunningBalanceMinor)
}

func TestHandlerCustomerLedgerRefusesABadNumber(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	handler := setupHandler(t, customerledgerportsmock.NewMockAccountsReceivableRepository(t))

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/accounts-receivable/customers/" + customerID.String() + "/ledger/").
		WithQuery(map[string]string{"first": "ten"}).
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}
