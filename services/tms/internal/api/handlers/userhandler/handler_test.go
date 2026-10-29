package userhandler_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/permchangetest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("user not found")

type setupOptions struct {
	userRepo    *tenantportsmock.MockUserRepository
	roleRepo    *permissionportsmock.MockRoleRepository
	sessionRepo *sessionportsmock.MockSessionRepository
	storage     *kernelportsmock.MockClient
	permEngine  permissionports.PermissionEngine
	cfg         *config.Config
}

func setupUserHandler(t *testing.T, opts setupOptions) *userhandler.Handler {
	t.Helper()

	if opts.userRepo == nil {
		opts.userRepo = tenantportsmock.NewMockUserRepository(t)
	}
	if opts.roleRepo == nil {
		opts.roleRepo = permissionportsmock.NewMockRoleRepository(t)
	}
	if opts.sessionRepo == nil {
		opts.sessionRepo = sessionportsmock.NewMockSessionRepository(t)
	}
	if opts.permEngine == nil {
		opts.permEngine = &mocks.AllowAllPermissionEngine{}
	}
	if opts.storage == nil {
		opts.storage = kernelportsmock.NewMockClient(t)
	}
	if opts.cfg == nil {
		opts.cfg = &config.Config{
			App: appconfig.Config{Debug: true},
			Security: securityconfig.Config{
				Session: securityconfig.SessionConfig{
					Name: "trenova_session",
				},
			},
			Storage: storageconfig.Config{
				MaxFileSize:        5 * 1024 * 1024,
				PresignedURLExpiry: 15 * time.Minute,
				AllowedMIMETypes:   []string{"image/jpeg", "image/png", "image/webp"},
			},
		}
	}

	logger := zap.NewNop()

	userSvc := userservice.New(userservice.Params{
		Logger:            logger,
		Repo:              opts.userRepo,
		SessionRepository: opts.sessionRepo,
		AuditService:      &mocks.NoopAuditService{},
		Realtime:          &mocks.NoopRealtimeService{},
		Storage:           opts.storage,
		StorageConfig:     &opts.cfg.Storage,
		Validator:         userservice.NewTestValidator(),
		PermissionEngine:  opts.permEngine,
	})

	roleSvc := roleservice.New(roleservice.Params{
		Logger:           logger,
		RoleRepo:         opts.roleRepo,
		Notifier:         &permchangetest.Recorder{},
		PermissionEngine: opts.permEngine,
		Validator:        roleservice.NewTestValidator(),
		Registry:         permissioncatalog.NewEmptyRegistry(),
	})

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &opts.cfg.App,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: opts.permEngine,
		ErrorHandler:     errorHandler,
	})

	return userhandler.New(userhandler.Params{
		Service:              userSvc,
		RoleService:          roleSvc,
		PermissionEngine:     opts.permEngine,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestUserHandler_List_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*tenant.User]{
		Items: []*tenant.User{
			{
				ID:                    userID,
				CurrentOrganizationID: testutil.TestOrgID,
				BusinessUnitID:        testutil.TestBuID,
				Name:                  "Test User",
				Username:              "testuser",
				EmailAddress:          "test@example.com",
				Status:                domaintypes.StatusActive,
			},
		},
		Total:      1,
		TotalExact: true,
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 1, resp.Count)
	assert.Len(t, resp.Results, 1)
}

func TestUserHandler_List_WithPagination(t *testing.T) {
	t.Parallel()

	const requestedLimit = 10

	items := make([]*tenant.User, 0, requestedLimit+1)
	for i := range requestedLimit + 1 {
		items = append(items, &tenant.User{
			ID:                    pulid.MustNew("usr_"),
			CurrentOrganizationID: testutil.TestOrgID,
			BusinessUnitID:        testutil.TestBuID,
			Name:                  fmt.Sprintf("User %02d", i),
			Username:              fmt.Sprintf("user%02d", i),
			EmailAddress:          fmt.Sprintf("user%02d@example.com", i),
			Status:                domaintypes.StatusActive,
			CreatedAt:             timeutils.NowUnix(),
		})
	}

	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.MatchedBy(func(req *tenantports.ListUsersRequest) bool {
		return req.Filter.Pagination.Limit == requestedLimit+1
	})).Return(&pagination.ListResult[*tenant.User]{
		Items:      items,
		Total:      50,
		TotalExact: true,
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithQuery(map[string]string{"limit": "10"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.CursorResponse[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 50, resp.Count)
	assert.Len(t, resp.Results, requestedLimit)
	assert.True(t, resp.HasNextPage)
	assert.NotEmpty(t, resp.EndCursor)
}

func TestUserHandler_List_RejectsOffset(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithQuery(map[string]string{"limit": "10", "offset": "20"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUserHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func makeUserRepo(t *testing.T) *tenantportsmock.MockUserRepository {
	t.Helper()
	repo := tenantportsmock.NewMockUserRepository(t)
	return repo
}

func makeUserRepoWithGetByID(
	t *testing.T,
	user *tenant.User,
	err error,
) *tenantportsmock.MockUserRepository {
	t.Helper()
	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(user, err)
	return repo
}

func TestUserHandler_Get_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := makeUserRepoWithGetByID(t, &tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Test User", resp["name"])
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := makeUserRepoWithGetByID(t, nil, errNotFound)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_Me_Success(t *testing.T) {
	t.Parallel()

	repo := makeUserRepoWithGetByID(t, &tenant.User{
		ID:                    testutil.TestUserID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Current User",
		Username:              "currentuser",
		EmailAddress:          "current@example.com",
		Status:                domaintypes.StatusActive,
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/me/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Current User", resp["name"])
}

func TestUserHandler_Me_Error(t *testing.T) {
	t.Parallel()

	repo := makeUserRepoWithGetByID(t, nil, errors.New("failed to get user"))

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/me/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_GetRoleAssignments_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	roleRepo := permissionportsmock.NewMockRoleRepository(t)
	roleRepo.ExpectedCalls = nil
	roleRepo.On("GetUserRoleAssignments", mock.Anything, userID, mock.Anything).
		Return([]*permission.UserRoleAssignment{
			{
				ID:     pulid.MustNew("ura_"),
				UserID: userID,
				RoleID: pulid.MustNew("rol_"),
			},
		}, nil)

	handler := setupUserHandler(t, setupOptions{roleRepo: roleRepo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/role-assignments/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_GetRoleAssignments_Error(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	roleRepo := permissionportsmock.NewMockRoleRepository(t)
	roleRepo.ExpectedCalls = nil
	roleRepo.On("GetUserRoleAssignments", mock.Anything, userID, mock.Anything).
		Return(nil, errors.New("role repo error"))

	handler := setupUserHandler(t, setupOptions{roleRepo: roleRepo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/role-assignments/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_GetEffectivePermissions_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/effective-permissions/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_GetEffectivePermissions_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/invalid-id/effective-permissions/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUserHandler_GetEffectivePermissions_EngineError(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	permEngine := permissionportsmock.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil)
	permEngine.On("GetEffectivePermissions", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("permission engine error"))

	handler := setupUserHandler(t, setupOptions{permEngine: permEngine})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/" + userID.String() + "/effective-permissions/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_BulkUpdateStatus_Success(t *testing.T) {
	t.Parallel()

	userID1 := pulid.MustNew("usr_")
	userID2 := pulid.MustNew("usr_")
	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return([]*tenant.User{
		{ID: userID1, Status: domaintypes.StatusActive},
		{ID: userID2, Status: domaintypes.StatusActive},
	}, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Return([]*tenant.User{
		{ID: userID1, Status: domaintypes.StatusInactive},
		{ID: userID2, Status: domaintypes.StatusInactive},
	}, nil)
	sessions := sessionportsmock.NewMockSessionRepository(t)
	sessions.On("DeleteAllForUser", mock.Anything, userID1).Return(nil).Once()
	sessions.On("DeleteAllForUser", mock.Anything, userID2).Return(nil).Once()

	handler := setupUserHandler(t, setupOptions{userRepo: repo, sessionRepo: sessions})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userIds": []string{userID1.String(), userID2.String()},
			"status":  "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_BulkUpdateStatus_BadJSON(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/bulk-update-status/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_BulkUpdateStatus_ServiceError(t *testing.T) {
	t.Parallel()

	userID1 := pulid.MustNew("usr_")
	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("GetByIDs", mock.Anything, mock.Anything).Return(nil, errors.New("get by ids failed"))

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/bulk-update-status/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"userIds": []string{userID1.String()},
			"status":  "Inactive",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_SelectOptions_Success(t *testing.T) {
	t.Parallel()

	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*tenant.User]{
			Items: []*tenant.User{
				{
					ID:   pulid.MustNew("usr_"),
					Name: "User One",
				},
			},
			Total:      1,
			TotalExact: true,
		}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_SelectOptions_Error(t *testing.T) {
	t.Parallel()

	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Return(nil, errors.New("select options error"))

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/select-options/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_GetOption_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := makeUserRepoWithGetByID(t, &tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Option User",
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/select-options/" + userID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Option User", resp["name"])
}

func TestUserHandler_GetOption_NotFound(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := makeUserRepoWithGetByID(t, nil, errNotFound)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/select-options/" + userID.String()).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_SimulatePermissions_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	addRoleID := pulid.MustNew("rol_")

	permEngine := permissionportsmock.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil)
	permEngine.On("SimulatePermissions", mock.Anything, mock.Anything).
		Return(&permissionports.EffectivePermissions{
			UserID:         userID,
			OrganizationID: testutil.TestOrgID,
		}, nil)

	handler := setupUserHandler(t, setupOptions{permEngine: permEngine})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/" + userID.String() + "/permissions/simulate/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"addRoleIds":    []string{addRoleID.String()},
			"removeRoleIds": []string{},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_SimulatePermissions_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/invalid-id/permissions/simulate/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"addRoleIds":    []string{},
			"removeRoleIds": []string{},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUserHandler_SimulatePermissions_BadJSON(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/" + userID.String() + "/permissions/simulate/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_SimulatePermissions_EngineError(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	permEngine := permissionportsmock.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil)
	permEngine.On("SimulatePermissions", mock.Anything, mock.Anything).
		Return(nil, errors.New("simulate error"))

	handler := setupUserHandler(t, setupOptions{permEngine: permEngine})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/" + userID.String() + "/permissions/simulate/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"addRoleIds":    []string{},
			"removeRoleIds": []string{},
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_GetOrganizations_Success(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("GetOrganizations", mock.Anything, mock.Anything).
		Return([]*tenant.OrganizationMembership{
			{
				ID:             pulid.MustNew("uom_"),
				OrganizationID: orgID,
				IsDefault:      true,
				Organization:   &tenant.Organization{Name: "Test Org"},
			},
		}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/me/organizations/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_GetOrganizations_Error(t *testing.T) {
	t.Parallel()

	repo := makeUserRepo(t)
	repo.ExpectedCalls = nil
	repo.On("GetOrganizations", mock.Anything, mock.Anything).
		Return(nil, errors.New("organizations fetch error"))

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/me/organizations/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_SwitchOrganization_Success(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	targetOrgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	sessionRepo := sessionportsmock.NewMockSessionRepository(t)
	sessionRepo.On("Get", mock.Anything, mock.Anything).Return(&session.Session{
		ID:             sessionID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
	}, nil)
	sessionRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	userRepo := tenantportsmock.NewMockUserRepository(t)
	userRepo.On("GetOrganizations", mock.Anything, mock.Anything).
		Return([]*tenant.OrganizationMembership{
			{
				ID:             pulid.MustNew("uom_"),
				OrganizationID: targetOrgID,
				BusinessUnitID: buID,
				IsDefault:      false,
				Organization:   &tenant.Organization{Name: "Target Org"},
			},
		}, nil)
	userRepo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.User{
		ID:                    testutil.TestUserID,
		CurrentOrganizationID: targetOrgID,
		BusinessUnitID:        buID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
	}, nil)
	userRepo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	handler := setupUserHandler(t, setupOptions{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultSessionAuthContext(sessionID).
		WithJSONBody(map[string]any{
			"organizationId": targetOrgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestUserHandler_SwitchOrganization_NoCookie(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"organizationId": pulid.MustNew("org_").String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_SwitchOrganization_RequiresASessionPrincipal(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"organizationId": pulid.MustNew("org_").String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_SwitchOrganization_BadJSON(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultSessionAuthContext(sessionID).
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_SwitchOrganization_InvalidOrgID(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultSessionAuthContext(sessionID).
		WithJSONBody(map[string]any{
			"organizationId": "not-a-valid-pulid",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_SwitchOrganization_ServiceError(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	targetOrgID := pulid.MustNew("org_")

	sessionRepo := sessionportsmock.NewMockSessionRepository(t)
	sessionRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("session not found"))

	handler := setupUserHandler(t, setupOptions{sessionRepo: sessionRepo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultSessionAuthContext(sessionID).
		WithJSONBody(map[string]any{
			"organizationId": targetOrgID.String(),
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_UpdateMySettings_Success(t *testing.T) {
	t.Parallel()

	user := &tenant.User{
		ID:                    testutil.TestUserID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Updated User",
		Username:              "updateduser",
		EmailAddress:          "updated@example.com",
		Timezone:              "America/Chicago",
		TimeFormat:            domaintypes.TimeFormat24Hour,
	}

	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(newTestUserForHandler(), nil).Once()
	repo.On("UpdateColumns", mock.Anything, mock.MatchedBy(func(req tenantports.UpdateUserColumnsRequest) bool {
		return assert.ObjectsAreEqual([]string{"timezone", "time_format"}, req.Columns) &&
			req.Entity.Timezone == "America/Chicago" &&
			req.Entity.TimeFormat == domaintypes.TimeFormat24Hour
	})).
		Return(user, nil).
		Once()
	repo.On("GetByID", mock.Anything, mock.Anything).Return(user, nil).Once()

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/me/settings/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"timezone":   "America/Chicago",
			"timeFormat": domaintypes.TimeFormat24Hour,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp tenant.User
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated User", resp.Name)
	assert.Equal(t, "updated@example.com", resp.EmailAddress)
}

func TestUserHandler_UpdateMySettings_RejectsAdminManagedFields(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/me/settings/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":       "Not Allowed",
			"username":   "notallowed",
			"timezone":   "America/Chicago",
			"timeFormat": domaintypes.TimeFormat24Hour,
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_UpdateMyAssistantVisibility_HidesTheAssistant(t *testing.T) {
	t.Parallel()

	hidden := newTestUserForHandler()
	hidden.AssistantHidden = true

	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(newTestUserForHandler(), nil).Once()
	repo.On("UpdateColumns", mock.Anything, mock.MatchedBy(func(req tenantports.UpdateUserColumnsRequest) bool {
		return assert.ObjectsAreEqual([]string{"assistant_hidden"}, req.Columns) &&
			req.Entity.AssistantHidden
	})).
		Return(hidden, nil).
		Once()
	repo.On("GetByID", mock.Anything, mock.Anything).Return(hidden, nil).Once()

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/me/assistant-visibility/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"assistantHidden": true})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, true, resp["assistantHidden"])
}

func TestUserHandler_UpdateMyAssistantVisibility_RequiresTheChoice(t *testing.T) {
	t.Parallel()

	repo := tenantportsmock.NewMockUserRepository(t)
	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/me/assistant-visibility/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
	repo.AssertNotCalled(t, "UpdateColumns", mock.Anything, mock.Anything)
}

func TestUserHandler_ChangeMyPassword_Success(t *testing.T) {
	t.Parallel()

	user := newTestUserForHandler()
	hashedPassword, err := user.GeneratePassword("current-password")
	require.NoError(t, err)
	user.Password = hashedPassword

	repo := tenantportsmock.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(user, nil).Twice()
	repo.On("UpdatePassword", mock.Anything, mock.Anything).Return(nil).Once()

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/change-password/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"currentPassword": "current-password",
			"newPassword":     "new-password",
			"confirmPassword": "new-password",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp tenant.User
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, user.ID, resp.ID)
	assert.Equal(t, user.EmailAddress, resp.EmailAddress)
}

func TestUserHandler_ChangeMyPassword_ValidationFailure(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/change-password/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"currentPassword": "same-password",
			"newPassword":     "same-password",
			"confirmPassword": "different-password",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func newTestUserForHandler() *tenant.User {
	return &tenant.User{
		ID:                    testutil.TestUserID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Timezone:              "America/New_York",
		TimeFormat:            domaintypes.TimeFormat12Hour,
		Status:                domaintypes.StatusActive,
	}
}

func TestUserHandler_SimulatePermissions_RequiresUserRead(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{
		permEngine: &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceUser.String()},
		},
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/" + pulid.MustNew("usr_").String() + "/permissions/simulate/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{"addRoleIds": []string{}, "removeRoleIds": []string{}})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}
