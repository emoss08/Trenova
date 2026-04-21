package accountsreceivablehandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/accountsreceivablehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupHandler(
	t *testing.T,
	repo repositories.AccountsReceivableRepository,
) *accountsreceivablehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	pm := middleware.NewPermissionMiddleware(
		middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.AllowAllPermissionEngine{},
			ErrorHandler:     errorHandler,
		},
	)
	service := accountsreceivableservice.New(
		accountsreceivableservice.Params{Logger: logger, Repo: repo},
	)

	return accountsreceivablehandler.New(
		accountsreceivablehandler.Params{
			Service:              service,
			ErrorHandler:         errorHandler,
			PermissionMiddleware: pm,
		},
	)
}

func TestHandlerOpenItems(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().
		ListOpenItems(mock.Anything, mock.Anything).
		Return([]*repositories.AROpenItem{
			{
				InvoiceID:       pulid.MustNew("inv_"),
				CustomerID:      pulid.MustNew("cus_"),
				OpenAmountMinor: 5000,
			},
		}, nil).
		Once()
	handler := setupHandler(t, repo)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/accounts-receivable/open-items/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp []*repositories.AROpenItem
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, int64(5000), resp[0].OpenAmountMinor)
}

func TestHandlerOpenItemsBadCustomerID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAccountsReceivableRepository(t)
	handler := setupHandler(t, repo)
	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/accounting/accounts-receivable/open-items/").
		WithQuery(map[string]string{"customerId": "bad-id"}).
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestHandlerCustomerStatement(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := mocks.NewMockAccountsReceivableRepository(t)
	repo.EXPECT().GetCustomerName(mock.Anything, mock.Anything).Return("Acme", nil).Once()
	repo.EXPECT().
		ListOpenItems(mock.Anything, mock.Anything).
		Return([]*repositories.AROpenItem{
			{InvoiceID: pulid.MustNew("inv_"), CustomerID: customerID, OpenAmountMinor: 5000},
		}, nil).
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
				AmountMinor:     -5000,
			},
		}, nil).
		Once()
	repo.EXPECT().
		GetCustomerAging(mock.Anything, mock.Anything).
		Return(&repositories.ARCustomerAgingRow{
			CustomerID:   customerID,
			CustomerName: "Acme",
			Buckets:      repositories.ARAgingBucketTotals{TotalOpenMinor: 5000},
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
	var resp serviceports.ARCustomerStatement
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, customerID, resp.CustomerID)
	assert.Equal(t, "Acme", resp.CustomerName)
	require.Len(t, resp.Transactions, 2)
}
