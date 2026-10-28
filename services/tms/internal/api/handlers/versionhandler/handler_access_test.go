package versionhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/versionhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Forcing an update check spends the instance's unauthenticated GitHub quota,
// so it is neither public nor open to every signed-in user.
func TestCheckUpdates_IsNotPublicAndNeedsOrganizationUpdate(t *testing.T) {
	t.Parallel()

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{},
	})
	handler := versionhandler.New(versionhandler.Params{
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
			PermissionEngine: &mocks.DenyResourcesPermissionEngine{
				Denied: []string{permission.ResourceOrganization.String()},
			},
			ErrorHandler: errorHandler,
		}),
	})

	public := gin.New()
	handler.RegisterPublicRoutes(public.Group("/api/v1"))
	for _, route := range public.Routes() {
		assert.NotEqual(t, "/api/v1/system/check-updates", route.Path)
		assert.NotEqual(t, "/api/v1/system/update-status", route.Path)
	}

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/system/check-updates").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}
