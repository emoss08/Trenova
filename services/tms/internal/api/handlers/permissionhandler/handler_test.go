package permissionhandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/permissionhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupPermissionHandler(
	t *testing.T,
	engine *mocks.MockPermissionEngine,
) *permissionhandler.Handler {
	t.Helper()

	logger := zap.NewNop()

	registry := permission.NewEmptyRegistry()
	_ = registry.Register(&permission.ResourceDefinition{
		Resource:    "test_resource",
		DisplayName: "Test Resource",
		Category:    "test",
	})

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})

	return permissionhandler.New(permissionhandler.Params{
		PermissionEngine: engine,
		Registry:         registry,
		ErrorHandler:     errorHandler,
	})
}

func TestPermissionHandler_GetManifest_Success(t *testing.T) {
	t.Parallel()

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("GetLightManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(&services.LightPermissionManifest{
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

	engine := mocks.NewMockPermissionEngine(t)
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

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("GetLightManifest", mock.Anything, mock.Anything, mock.Anything).
		Return(&services.LightPermissionManifest{
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

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("GetResourcePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&services.ResourcePermissionDetail{
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

	engine := mocks.NewMockPermissionEngine(t)
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

	engine := mocks.NewMockPermissionEngine(t)
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

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("CheckBatch", mock.Anything, mock.Anything).
		Return(&services.BatchPermissionCheckResult{
			Results: []services.PermissionCheckResult{
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

	var resp services.BatchPermissionCheckResult
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp.Results, 1)
}

func TestPermissionHandler_CheckBatch_BadJSON(t *testing.T) {
	t.Parallel()

	engine := mocks.NewMockPermissionEngine(t)
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

	engine := mocks.NewMockPermissionEngine(t)
	handler := setupPermissionHandler(t, engine)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/permissions/resources").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "test", resp[0]["category"])
}

func TestPermissionHandler_GetAvailableOperations_Success(t *testing.T) {
	t.Parallel()

	engine := mocks.NewMockPermissionEngine(t)
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
