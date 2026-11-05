package exchangeratehandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/exchangeratehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/exchangerate"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

const (
	refreshPath = "/api/v1/exchange-rates/refresh"
	quotePath   = "/api/v1/exchange-rates/settlement-quotes"
	latestPath  = "/api/v1/exchange-rates/latest"
	convertPath = "/api/v1/exchange-rates/convert"
)

func serve(
	t *testing.T,
	service *exchangerateportsmock.MockExchangeRateService,
	engine *permissionportsmock.MockPermissionEngine,
	path string,
) int {
	t.Helper()

	return serveMethod(t, service, engine, http.MethodPost, path)
}

func serveMethod(
	t *testing.T,
	service *exchangerateportsmock.MockExchangeRateService,
	engine *permissionportsmock.MockPermissionEngine,
	method, path string,
) int {
	t.Helper()

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{Debug: true},
	})
	handler := exchangeratehandler.New(exchangeratehandler.Params{
		ExchangeRateService: service,
		ErrorHandler:        errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: engine,
				ErrorHandler:     errorHandler,
			},
		),
	})

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(method).
		WithPath(path).
		WithQuery(map[string]string{"from": "USD", "to": "CAD", "amount": "10"}).
		WithDefaultAuthContext().
		WithJSONBody(map[string]string{
			"fromCurrency": "USD",
			"toCurrency":   "CAD",
			"amount":       "10",
		})
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	return ginCtx.ResponseCode()
}

func expectCheck(
	engine *permissionportsmock.MockPermissionEngine,
	resource permission.Resource,
	operation permission.Operation,
	allowed bool,
) {
	engine.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == resource.String() && req.Operation == operation
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: allowed}, nil).
		Once()
}

func TestRefresh_TakesIntegrationUpdate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceIntegration, permission.OpUpdate, true)
	service.EXPECT().RefreshRates(mock.Anything, mock.Anything, "USD").Return(nil).Once()

	assert.Equal(t, http.StatusOK, serve(t, service, engine, refreshPath))
}

func TestRefresh_RefusedWithoutIntegrationUpdate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceIntegration, permission.OpUpdate, false)

	assert.Equal(t, http.StatusForbidden, serve(t, service, engine, refreshPath))
}

func TestSettlementQuote_TakesCustomerPaymentCreate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceCustomerPayment, permission.OpCreate, true)
	service.EXPECT().
		CreateSettlementQuote(mock.Anything, mock.Anything, mock.Anything).
		Return(&exchangerate.SettlementQuote{}, nil).
		Once()

	assert.Equal(t, http.StatusCreated, serve(t, service, engine, quotePath))
}

func TestSettlementQuote_RefusedWithoutCustomerPaymentCreate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceCustomerPayment, permission.OpCreate, false)

	assert.Equal(t, http.StatusForbidden, serve(t, service, engine, quotePath))
}

func TestLatest_TakesIntegrationUpdate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceIntegration, permission.OpUpdate, true)
	service.EXPECT().
		GetLatestRates(mock.Anything, mock.Anything, "USD").
		Return(&exchangerateports.LatestRatesResult{}, nil).
		Once()

	assert.Equal(t, http.StatusOK, serveMethod(t, service, engine, http.MethodGet, latestPath))
}

func TestLatest_RefusedWithoutIntegrationUpdate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceIntegration, permission.OpUpdate, false)

	assert.Equal(
		t,
		http.StatusForbidden,
		serveMethod(t, service, engine, http.MethodGet, latestPath),
	)
}

func TestConvert_TakesCustomerPaymentCreate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceCustomerPayment, permission.OpCreate, true)
	service.EXPECT().
		Convert(mock.Anything, mock.Anything, "USD", "CAD", mock.Anything, mock.Anything).
		Return(&exchangerateports.RateConversionResult{}, nil).
		Once()

	assert.Equal(t, http.StatusOK, serveMethod(t, service, engine, http.MethodGet, convertPath))
}

func TestConvert_RefusedWithoutCustomerPaymentCreate(t *testing.T) {
	t.Parallel()

	service := exchangerateportsmock.NewMockExchangeRateService(t)
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, permission.ResourceCustomerPayment, permission.OpCreate, false)

	assert.Equal(
		t,
		http.StatusForbidden,
		serveMethod(t, service, engine, http.MethodGet, convertPath),
	)
}
