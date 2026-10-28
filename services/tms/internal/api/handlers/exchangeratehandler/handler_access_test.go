package exchangeratehandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/exchangeratehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Refreshing rates and quoting a settlement call the paid rate provider, and
// a quote is stored. A driver signed in to the portal is not someone who
// does either.
func TestExchangeRateRoutes_NeedFinanceAccess(t *testing.T) {
	t.Parallel()

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/exchange-rates/convert"},
		{http.MethodGet, "/api/v1/exchange-rates/latest"},
		{http.MethodPost, "/api/v1/exchange-rates/refresh"},
		{http.MethodPost, "/api/v1/exchange-rates/settlement-quotes"},
	} {
		errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: &config.Config{},
		})
		handler := exchangeratehandler.New(exchangeratehandler.Params{
			ErrorHandler: errorHandler,
			PermissionMiddleware: middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.DenyResourcesPermissionEngine{Denied: []string{
					permission.ResourceCustomerPayment.String(),
					permission.ResourceInvoice.String(),
					permission.ResourceIntegration.String(),
				}},
				ErrorHandler: errorHandler,
			}),
		})

		ginCtx := testutil.NewGinTestContext().
			WithMethod(route.method).
			WithPath(route.path).
			WithDefaultAuthContext()
		handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
		ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

		assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode(), route.path)
	}
}
