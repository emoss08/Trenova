package accountsreceivablehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/accountsreceivablehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupHandler(t *testing.T, repo repositories.AccountsReceivableRepository) *accountsreceivablehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{PermissionEngine: &mocks.AllowAllPermissionEngine{}, ErrorHandler: errorHandler})
	service := accountsreceivableservice.New(accountsreceivableservice.Params{Logger: logger, Repo: repo})

	return accountsreceivablehandler.New(accountsreceivablehandler.Params{Service: service, ErrorHandler: errorHandler, PermissionMiddleware: pm})
}

func TestHandlerOpenItems(t *testing.T) {
	t.Parallel()

	repo := fakeARRepo{items: []*accountsreceivable.OpenItem{{InvoiceID: pulid.MustNew("inv_"), CustomerID: pulid.MustNew("cus_"), OpenAmountMinor: 5000}}}
	handler := setupHandler(t, repo)

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/accounts-receivable/open-items/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp []*accountsreceivable.OpenItem
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, int64(5000), resp[0].OpenAmountMinor)
}

func TestHandlerOpenItemsBadCustomerID(t *testing.T) {
	t.Parallel()

	handler := setupHandler(t, fakeARRepo{})
	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/accounts-receivable/open-items/").WithQuery(map[string]string{"customerId": "bad-id"}).WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestHandlerCustomerStatement(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	repo := fakeARRepo{
		name:  "Acme",
		items: []*accountsreceivable.OpenItem{{InvoiceID: pulid.MustNew("inv_"), CustomerID: customerID, OpenAmountMinor: 5000}},
		ledger: []*accountsreceivable.LedgerEntry{
			{CustomerID: customerID, TransactionDate: 10, EventType: "InvoicePosted", DocumentNumber: "INV-1", AmountMinor: 10000},
			{CustomerID: customerID, TransactionDate: 20, EventType: "CustomerPaymentPosted", DocumentNumber: "PAY-1", AmountMinor: -5000},
		},
		aging: &accountsreceivable.CustomerAgingRow{CustomerID: customerID, CustomerName: "Acme", Buckets: accountsreceivable.AgingBucketTotals{TotalOpenMinor: 5000}},
	}
	handler := setupHandler(t, repo)

	ginCtx := sharedtestutil.NewGinTestContext().WithMethod(http.MethodGet).WithPath("/api/v1/accounting/accounts-receivable/customers/" + customerID.String() + "/statement/").WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp accountsreceivable.CustomerStatement
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, customerID, resp.CustomerID)
	assert.Equal(t, "Acme", resp.CustomerName)
	require.Len(t, resp.Transactions, 2)
}

type fakeARRepo struct {
	ledger []*accountsreceivable.LedgerEntry
	rows   []*accountsreceivable.CustomerAgingRow
	items  []*accountsreceivable.OpenItem
	name   string
	aging  *accountsreceivable.CustomerAgingRow
}

func (f fakeARRepo) ListCustomerLedger(_ context.Context, _ repositories.ListCustomerLedgerRequest) ([]*accountsreceivable.LedgerEntry, error) {
	return f.ledger, nil
}

func (f fakeARRepo) ListARAging(_ context.Context, _ repositories.ListARAgingRequest) ([]*accountsreceivable.CustomerAgingRow, error) {
	return f.rows, nil
}

func (f fakeARRepo) ListOpenItems(_ context.Context, _ repositories.ListAROpenItemsRequest) ([]*accountsreceivable.OpenItem, error) {
	return f.items, nil
}

func (f fakeARRepo) GetCustomerName(_ context.Context, _ repositories.GetARCustomerNameRequest) (string, error) {
	return f.name, nil
}

func (f fakeARRepo) GetCustomerAging(_ context.Context, _ repositories.GetARCustomerAgingRequest) (*accountsreceivable.CustomerAgingRow, error) {
	return f.aging, nil
}
