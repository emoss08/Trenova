package permissionhandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"

	"github.com/emoss08/trenova/internal/api/handlers/permissionhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/scopepolicytest"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupPermissionHandler(
	t *testing.T,
	engine *permissionportsmock.MockPermissionEngine,
) *permissionhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	registry := permission.NewEmptyRegistry()
	_ = registry.Register(&permission.ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
		Category:    "test",
	})
	_ = registry.Register(&permission.ResourceDefinition{
		Resource:    "test_owned",
		DisplayName: "Test Owned",
		Category:    "test",
		Scopes: []permission.ScopeSupport{
			{Scope: permission.DataScopeOwn, EnforcedBy: permission.ScopeEnforcedByDatabase},
		},
	})

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})

	return permissionhandler.New(permissionhandler.Params{
		PermissionEngine: engine,
		Registry:         registry,
		Scopes:           &scopepolicytest.Policy{Registry: registry, DatabaseEnforcing: true},
		ErrorHandler:     errorHandler,
	})
}

func TestPermissionHandler_GetManifest_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetLightManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(&permissionports.LightPermissionManifest{
			Checksum:  "abc123",
			ExpiresAt: 1700000000,
		}, nil)

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "abc123", resp["checksum"])
}

func TestPermissionHandler_GetManifest_Error(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetLightManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("internal error"))

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestPermissionHandler_GetVersion_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetLightManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(&permissionports.LightPermissionManifest{
			Checksum:  "abc123",
			ExpiresAt: 1700000000,
		}, nil)

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions/version").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "abc123", resp["checksum"])
}

func TestPermissionHandler_GetResourcePermissions_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetResourcePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&permissionports.ResourcePermissionDetail{
			Resource:   "test_resource",
			Operations: []permission.Operation{permission.OpRead},
		}, nil)

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions/test_resource").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "test_resource", resp["resource"])
}

func TestPermissionHandler_GetResourcePermissions_NotFound(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetResourcePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil)

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions/nonexistent").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
}

func TestPermissionHandler_GetResourcePermissions_Error(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("GetResourcePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("internal error"))

	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/me/permissions/test_resource").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestPermissionHandler_CheckBatch_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("CheckBatch", mock.Anything, mock.Anything).
		Return(&permissionports.BatchPermissionCheckResult{
			Results: []permissionports.PermissionCheckResult{
				{Allowed: true},
			},
		}, nil)

	handler := setupPermissionHandler(t, engine)

	body := map[string]any{
		"checks": []map[string]any{
			{"resource": "test_resource", "operation": "read"},
		},
	}

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/me/permissions/check").
		WithJSONBody(body).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp permissionports.BatchPermissionCheckResult
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp.Results, 1)
}

func TestPermissionHandler_CheckBatch_BadJSON(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/me/permissions/check").
		WithBody("{invalid json").
		WithHeader("Content-Type", "application/json").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.GreaterOrEqual(t, ginCtx.ResponseCode(), 400)
}

func TestPermissionHandler_GetAvailableResources_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/permissions/resources").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []struct {
		Category  string `json:"category"`
		Resources []struct {
			Resource          string   `json:"resource"`
			DisplayName       string   `json:"displayName"`
			EnforceableScopes []string `json:"enforceableScopes"`
		} `json:"resources"`
	}
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "test", resp[0].Category)
	require.Len(t, resp[0].Resources, 2)
	assert.Equal(t, "test_owned", resp[0].Resources[0].Resource)
	assert.Equal(t, "Test Owned", resp[0].Resources[0].DisplayName)
	assert.Equal(t, []string{"own", "organization"}, resp[0].Resources[0].EnforceableScopes)
	assert.Equal(t, []string{"organization"}, resp[0].Resources[1].EnforceableScopes)
}

func TestPermissionHandler_GetAvailableOperations_Success(t *testing.T) {
	t.Parallel()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/permissions/operations").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.NotEmpty(t, resp)
}

func TestPermissionHandler_GetBundles_DowngradesScopesTheDeploymentDoesNotEnforce(t *testing.T) {
	t.Parallel()

	logger := zap.NewNop()
	registry := permission.NewRegistry()
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	handler := permissionhandler.New(permissionhandler.Params{
		PermissionEngine: permissionportsmock.NewMockPermissionEngine(t),
		Registry:         registry,
		Scopes:           &scopepolicytest.Policy{Registry: registry},
		ErrorHandler: helpers.NewErrorHandler(
			helpers.ErrorHandlerParams{Logger: logger, Config: cfg},
		),
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/permissions/bundles").
		WithDefaultAuthContext()
	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	type grant struct {
		Resource    string   `json:"resource"`
		DisplayName string   `json:"displayName"`
		Operations  []string `json:"operations"`
		DataScope   string   `json:"dataScope"`
	}
	var resp struct {
		Bundles []struct {
			ID     string  `json:"id"`
			Grants []grant `json:"grants"`
		} `json:"bundles"`
		Templates []struct {
			ID     string  `json:"id"`
			Grants []grant `json:"grants"`
		} `json:"templates"`
	}
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp.Bundles, len(permission.Bundles()))
	require.Len(t, resp.Templates, len(permission.JobTemplates()))

	for _, bundle := range resp.Bundles {
		if bundle.ID != "hr.team" {
			continue
		}
		for _, g := range bundle.Grants {
			switch g.Resource {
			case "worker":
				assert.Equal(t, "organization", g.DataScope,
					"the database does not enforce team here, so the grant says organization")
				assert.Equal(t, "Worker", g.DisplayName)
			case "timesheet":
				assert.Equal(t, "team", g.DataScope, "the application enforces team on timesheets")
			}
		}
	}
}

func TestPermissionHandler_GetAvailableResources_TranslatesAndFollowsTheRail(t *testing.T) {
	t.Parallel()

	registry := permission.NewEmptyRegistry()
	require.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:    "invoice",
		DisplayName: "Invoice",
		Category:    "Billing",
		Operations: []permission.OperationDefinition{
			{Operation: permission.OpRead, DisplayName: "View"},
		},
	}))
	require.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:    "assistant",
		DisplayName: "Desk",
		Category:    "Desk",
	}))

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	handler := permissionhandler.New(permissionhandler.Params{
		PermissionEngine: permissionportsmock.NewMockPermissionEngine(t),
		Registry:         registry,
		Scopes:           &scopepolicytest.Policy{Registry: registry, DatabaseEnforcing: true},
		ErrorHandler: helpers.NewErrorHandler(
			helpers.ErrorHandlerParams{Logger: zap.NewNop(), Config: cfg},
		),
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/permissions/resources").
		WithDefaultAuthContext()
	ginCtx.Context.Request = ginCtx.Context.Request.WithContext(
		i18n.WithLocale(ginCtx.Context.Request.Context(), i18n.ES),
	)

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)
	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []struct {
		Category  string `json:"category"`
		Resources []struct {
			DisplayName string `json:"displayName"`
			Operations  []struct {
				DisplayName string `json:"displayName"`
			} `json:"operations"`
		} `json:"resources"`
	}
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 2)
	assert.Equal(t, i18n.Translate(i18n.ES, "Desk"), resp[0].Category)
	assert.Equal(t, "Facturación", resp[1].Category)
	require.Len(t, resp[1].Resources, 1)
	assert.Equal(t, "Factura", resp[1].Resources[0].DisplayName)
	require.Len(t, resp[1].Resources[0].Operations, 1)
	assert.Equal(t, "Ver", resp[1].Resources[0].Operations[0].DisplayName)

	def, ok := registry.Get("invoice")
	require.True(t, ok)
	assert.Equal(t, "Invoice", def.DisplayName, "translating a response must not change the registry")
}
