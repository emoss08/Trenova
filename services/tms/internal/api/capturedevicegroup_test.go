//revive:disable-next-line:var-naming
package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/capturehandler"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingReadScopes struct {
	principal *permissionports.PermissionCheckRequest
}

func (s *recordingReadScopes) Resolver(
	principal *permissionports.PermissionCheckRequest,
) readscope.Resolver {
	s.principal = principal
	return func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{}, nil
	}
}

func TestCaptureDeviceGroup_ReadsUnderTheScopeOfThePersonWhoPairedIt(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	scopes := &recordingReadScopes{}
	router := &Router{
		captureHandler: &capturehandler.Handler{},
		readOnlyGuard:  &middleware.ReadOnlyGuard{},
		readScopeMiddleware: middleware.NewReadScopeMiddleware(
			middleware.ReadScopeMiddlewareParams{Scopes: scopes},
		),
	}
	group := router.captureDeviceGroup(gin.New().Group(""))
	require.Len(t, group.Handlers, 3)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/capture/device/", http.NoBody)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	authctx.SetCaptureDeviceContext(
		c,
		pulid.MustNew("cdv_"),
		userID,
		pulid.MustNew("bu_"),
		orgID,
		nil,
	)

	group.Handlers[1](c)

	require.NotNil(t, scopes.principal, "the device's identity is known by the second handler")
	assert.Equal(t, userID, scopes.principal.UserID)
	assert.Equal(t, orgID, scopes.principal.OrganizationID)
	assert.NotNil(t, readscope.Current(c.Request.Context()))
}
