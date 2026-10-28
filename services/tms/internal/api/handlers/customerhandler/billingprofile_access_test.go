package customerhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/customerhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// A customer's billing profile carries its credit limit, balance and hold
// reason. Signing in is not enough to read it.
func TestBillingProfile_NeedsCustomerOrShipmentRead(t *testing.T) {
	t.Parallel()

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{},
	})
	handler := customerhandler.New(customerhandler.Params{
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.DenyResourcesPermissionEngine{Denied: []string{
				permission.ResourceCustomer.String(),
				permission.ResourceShipment.String(),
			}},
			ErrorHandler: errorHandler,
		}),
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/customers/" + pulid.MustNew("cus_").String() + "/billing-profile/").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}
