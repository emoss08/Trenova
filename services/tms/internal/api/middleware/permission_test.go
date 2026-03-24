package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockPermissionEngine struct {
	mock.Mock
}

func (m *mockPermissionEngine) Check(
	ctx context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.PermissionCheckResult), args.Error(1)
}

func (m *mockPermissionEngine) CheckBatch(
	ctx context.Context,
	req *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.BatchPermissionCheckResult), args.Error(1)
}

func (m *mockPermissionEngine) GetLightManifest(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.LightPermissionManifest, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.LightPermissionManifest), args.Error(1)
}

func (m *mockPermissionEngine) GetResourcePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
	resource string,
) (*services.ResourcePermissionDetail, error) {
	args := m.Called(ctx, userID, orgID, resource)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ResourcePermissionDetail), args.Error(1)
}

func (m *mockPermissionEngine) InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error {
	args := m.Called(ctx, userID, orgID)
	return args.Error(0)
}

func (m *mockPermissionEngine) GetEffectivePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.EffectivePermissions, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.EffectivePermissions), args.Error(1)
}

func (m *mockPermissionEngine) SimulatePermissions(
	ctx context.Context,
	req *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.EffectivePermissions), args.Error(1)
}

func newTestErrorHandler() *helpers.ErrorHandler {
	return helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: &config.Config{App: config.AppConfig{Debug: true}},
	})
}

func newPermissionMiddleware(engine *mockPermissionEngine) *PermissionMiddleware {
	return NewPermissionMiddleware(PermissionMiddlewareParams{
		PermissionEngine: engine,
		ErrorHandler:     newTestErrorHandler(),
	})
}

func setAuthMiddleware(userID, buID, orgID pulid.ID) gin.HandlerFunc {
	return func(c *gin.Context) {
		authctx.SetAuthContext(c, userID, buID, orgID)
		c.Next()
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *services.PermissionCheckRequest) bool {
		return req.UserID == userID && req.OrganizationID == orgID && req.Resource == "shipment" &&
			req.Operation == permission.OpRead
	})).
		Return(&services.PermissionCheckResult{Allowed: true}, nil)

	pm := newPermissionMiddleware(engine)

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequirePermission("shipment", permission.OpRead),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handlerCalled)
	engine.AssertExpectations(t)
}

func TestRequirePermission_Denied(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&services.PermissionCheckResult{Allowed: false}, nil)

	pm := newPermissionMiddleware(engine)

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequirePermission("shipment", permission.OpRead),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.NotEqual(t, http.StatusOK, w.Code)
	engine.AssertExpectations(t)
}

func TestRequirePermission_EngineError(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).Return(nil, errors.New("engine failure"))

	pm := newPermissionMiddleware(engine)

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequirePermission("shipment", permission.OpCreate),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.NotEqual(t, http.StatusOK, w.Code)
	require.NotEmpty(t, w.Body.String())
	engine.AssertExpectations(t)
}

func TestRequireAnyPermission_OneAllowed(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *services.PermissionCheckRequest) bool {
		return req.Resource == "shipment" && req.Operation == permission.OpRead
	})).
		Return(&services.PermissionCheckResult{Allowed: false}, nil)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *services.PermissionCheckRequest) bool {
		return req.Resource == "order" && req.Operation == permission.OpRead
	})).
		Return(&services.PermissionCheckResult{Allowed: true}, nil)

	pm := newPermissionMiddleware(engine)

	checks := []struct {
		Resource  string
		Operation permission.Operation
	}{
		{"shipment", permission.OpRead},
		{"order", permission.OpRead},
	}

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequireAnyPermission(checks...),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	engine.AssertExpectations(t)
}

func TestRequireAnyPermission_AllDenied(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&services.PermissionCheckResult{Allowed: false}, nil)

	pm := newPermissionMiddleware(engine)

	checks := []struct {
		Resource  string
		Operation permission.Operation
	}{
		{"shipment", permission.OpRead},
		{"order", permission.OpRead},
	}

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequireAnyPermission(checks...),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.NotEqual(t, http.StatusOK, w.Code)
	engine.AssertExpectations(t)
}

func TestRequireAllPermissions_AllAllowed(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&services.PermissionCheckResult{Allowed: true}, nil)

	pm := newPermissionMiddleware(engine)

	checks := []struct {
		Resource  string
		Operation permission.Operation
	}{
		{"shipment", permission.OpRead},
		{"order", permission.OpCreate},
	}

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequireAllPermissions(checks...),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
	engine.AssertExpectations(t)
}

func TestRequireAllPermissions_OneDenied(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *services.PermissionCheckRequest) bool {
		return req.Resource == "shipment"
	})).
		Return(&services.PermissionCheckResult{Allowed: true}, nil)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *services.PermissionCheckRequest) bool {
		return req.Resource == "order"
	})).
		Return(&services.PermissionCheckResult{Allowed: false}, nil)

	pm := newPermissionMiddleware(engine)

	checks := []struct {
		Resource  string
		Operation permission.Operation
	}{
		{"shipment", permission.OpRead},
		{"order", permission.OpCreate},
	}

	handlerCalled := false
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequireAllPermissions(checks...),
		func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.NotEqual(t, http.StatusOK, w.Code)
	engine.AssertExpectations(t)
}

func TestGetPermissionResult(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	expected := &services.PermissionCheckResult{
		Allowed:   true,
		DataScope: permission.DataScopeOrganization,
	}

	engine := new(mockPermissionEngine)
	engine.On("Check", mock.Anything, mock.Anything).Return(expected, nil)

	pm := newPermissionMiddleware(engine)

	var result *services.PermissionCheckResult
	r := gin.New()
	r.GET(
		"/test",
		setAuthMiddleware(userID, buID, orgID),
		pm.RequirePermission("shipment", permission.OpRead),
		func(c *gin.Context) {
			result = GetPermissionResult(c)
			c.Status(http.StatusOK)
		},
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	require.NotNil(t, result)
	assert.True(t, result.Allowed)
	assert.Equal(t, permission.DataScopeOrganization, result.DataScope)
	engine.AssertExpectations(t)
}
