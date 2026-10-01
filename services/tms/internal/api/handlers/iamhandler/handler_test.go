package iamhandler_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/iamhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func newIAMHandler(t *testing.T) *iamhandler.Handler {
	t.Helper()

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})

	return iamhandler.New(iamhandler.Params{
		Service:      mocks.NewMockIAMService(t),
		ErrorHandler: errorHandler,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{
				PermissionEngine: &mocks.AllowAllPermissionEngine{},
				ErrorHandler:     errorHandler,
			},
		),
	})
}

func TestIAMRoutes_RefuseAnOrganizationOtherThanTheSessions(t *testing.T) {
	t.Parallel()

	otherOrg := pulid.MustNew("org_").String()
	cases := []struct {
		method string
		path   string
		body   map[string]any
	}{
		{method: http.MethodGet, path: "identity-providers"},
		{method: http.MethodPost, path: "identity-providers", body: map[string]any{"name": "Okta"}},
		{method: http.MethodPost, path: "scim-directories", body: map[string]any{"name": "Directory"}},
		{method: http.MethodPost, path: "access-policies", body: map[string]any{"name": "Policy"}},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			handler := newIAMHandler(t)
			ginCtx := testutil.NewGinTestContext().
				WithMethod(tc.method).
				WithPath("/api/v1/organizations/" + otherOrg + "/iam/" + tc.path).
				WithDefaultAuthContext()
			if tc.body != nil {
				ginCtx = ginCtx.WithJSONBody(tc.body)
			}

			handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
			ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

			assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
		})
	}
}
