package rolehandler_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/api/handlers/rolehandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("role not found")

func setupRoleHandler(t *testing.T, repo *mocks.MockRoleRepository) *rolehandler.Handler {
	t.Helper()

	logger := zap.NewNop()
	permEngine := &mocks.AllowAllPermissionEngine{}

	userRepo := mocks.NewMockUserRepository(t)
	userRepo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[any]{}, nil)
	userRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("SelectOptions", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	userRepo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	userRepo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	userRepo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).
		Maybe().
		Return(nil, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	permCacheRepo := mocks.NewMockPermissionCacheRepository(t)
	permCacheRepo.On("Get", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	permCacheRepo.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	permCacheRepo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	permCacheRepo.On("InvalidateByRole", mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	permCacheRepo.On("InvalidateOrganization", mock.Anything, mock.Anything).Maybe().Return(nil)

	service := roleservice.New(roleservice.Params{
		Logger:           logger,
		RoleRepo:         repo,
		UserRepo:         userRepo,
		PermissionCache:  permCacheRepo,
		PermissionEngine: permEngine,
		Validator:        roleservice.NewTestValidator(),
		Registry:         permission.NewEmptyRegistry(),
	})

	cfg := &config.Config{
		App: config.AppConfig{
			Debug: true,
		},
	}

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	return rolehandler.New(rolehandler.Params{
		Service:          service,
		PermissionEngine: permEngine,
		ErrorHandler:     errorHandler,
	})
}

func TestRoleHandler_List_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*permission.Role]{
		Items: []*permission.Role{
			{
				ID:             roleID,
				OrganizationID: testutil.TestOrgID,
				Name:           "Admin",
				Description:    "Administrator role",
			},
		},
		Total: 1,
	}, nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestRoleHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*permission.Role]{
		Items: []*permission.Role{},
		Total: 25,
	}, nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/").
		WithQuery(map[string]string{"limit": "10", "offset": "0"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 25, resp.Count)
}

func TestRoleHandler_List_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestRoleHandler_Get_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Admin",
		Description:    "Administrator role",
	}, nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/" + roleID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Admin", resp["name"])
}

func TestRoleHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, errNotFound)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/" + roleID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestRoleHandler_Get_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_Create_Success(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":        "Editor",
			"description": "Editor role",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Editor", resp["name"])
}

func TestRoleHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_Create_ServiceError(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":        "Editor",
			"description": "Editor role",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_Update_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":        "Updated Editor",
			"description": "Updated editor role",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated Editor", resp["name"])
}

func TestRoleHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Updated",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String()).
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":        "Updated",
			"description": "Updated role",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_GetImpact_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetUsersWithRole", mock.Anything, mock.Anything).Return([]repositories.ImpactedUser{
		{
			UserID:         pulid.MustNew("usr_"),
			UserName:       "John Doe",
			OrganizationID: testutil.TestOrgID,
			OrgName:        "Test Org",
			AssignmentType: "direct",
		},
		{
			UserID:         pulid.MustNew("usr_"),
			UserName:       "Jane Smith",
			OrganizationID: testutil.TestOrgID,
			OrgName:        "Test Org",
			AssignmentType: "inherited",
		},
	}, nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/" + roleID.String() + "/impact").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Len(t, resp, 2)
	assert.Equal(t, "John Doe", resp[0]["userName"])
}

func TestRoleHandler_GetImpact_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/invalid-id/impact").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_GetImpact_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetUsersWithRole", mock.Anything, mock.Anything).
		Return(nil, errors.New("failed to get impacted users"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/" + roleID.String() + "/impact").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestRoleHandler_GetImpact_EmptyResult(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetUsersWithRole", mock.Anything, mock.Anything).
		Return([]repositories.ImpactedUser{}, nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/roles/" + roleID.String() + "/impact").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp []map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Empty(t, resp)
}

func TestRoleHandler_AddPermission_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("CreateResourcePermission", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read", "create"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "shipments", resp["resource"])
	assert.Equal(t, roleID.String(), resp["roleId"])
}

func TestRoleHandler_AddPermission_InvalidRoleID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/invalid-id/permissions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_AddPermission_BadJSON(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_AddPermission_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("CreateResourcePermission", mock.Anything, mock.Anything).
		Return(errors.New("failed to create permission"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_UpdatePermission_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("UpdateResourcePermission", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/" + permID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read", "update", "delete"},
			"dataScope":  "all",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "shipments", resp["resource"])
	assert.Equal(t, roleID.String(), resp["roleId"])
	assert.Equal(t, permID.String(), resp["id"])
}

func TestRoleHandler_UpdatePermission_InvalidRoleID(t *testing.T) {
	t.Parallel()

	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/invalid-id/permissions/" + permID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_UpdatePermission_InvalidPermID(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/invalid-id").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_UpdatePermission_BadJSON(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/" + permID.String()).
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_UpdatePermission_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("UpdateResourcePermission", mock.Anything, mock.Anything).
		Return(errors.New("update permission failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/" + permID.String()).
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"resource":   "shipments",
			"operations": []string{"read"},
			"dataScope":  "organization",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_DeletePermission_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("DeleteResourcePermission", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/" + permID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestRoleHandler_DeletePermission_InvalidRoleID(t *testing.T) {
	t.Parallel()

	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/invalid-id/permissions/" + permID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_DeletePermission_InvalidPermID(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_DeletePermission_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("DeleteResourcePermission", mock.Anything, mock.Anything).
		Return(errors.New("delete permission failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/" + roleID.String() + "/permissions/" + permID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_AssignRole_Success(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("CreateAssignment", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/assignments").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userId": userID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, userID.String(), resp["userId"])
	assert.Equal(t, roleID.String(), resp["roleId"])
}

func TestRoleHandler_AssignRole_WithExpiry(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	userID := pulid.MustNew("usr_")
	expiresAt := int64(1700000000)
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("CreateAssignment", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/assignments").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userId":    userID.String(),
			"expiresAt": expiresAt,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, userID.String(), resp["userId"])
	assert.Equal(t, float64(expiresAt), resp["expiresAt"])
}

func TestRoleHandler_AssignRole_InvalidRoleID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/invalid-id/assignments").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userId": pulid.MustNew("usr_").String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_AssignRole_BadJSON(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/assignments").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_AssignRole_ServiceError(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: testutil.TestOrgID,
		Name:           "Editor",
	}, nil)
	repo.On("CreateAssignment", mock.Anything, mock.Anything).
		Return(errors.New("assignment failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/roles/" + roleID.String() + "/assignments").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userId": userID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestRoleHandler_UnassignRole_Success(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("DeleteAssignment", mock.Anything, mock.Anything).Return(nil)

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/assignments/" + assignmentID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusNoContent, ginCtx.ResponseCode())
}

func TestRoleHandler_UnassignRole_InvalidID(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockRoleRepository(t)
	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/assignments/invalid-id").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestRoleHandler_UnassignRole_ServiceError(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.MustNew("ura_")
	repo := mocks.NewMockRoleRepository(t)
	repo.On("DeleteAssignment", mock.Anything, mock.Anything).Return(errors.New("unassign failed"))

	handler := setupRoleHandler(t, repo)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodDelete).
		WithPath("/api/v1/roles/assignments/" + assignmentID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}
