package userhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/roleservice"
	"github.com/emoss08/trenova/internal/core/services/userservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errNotFound = errors.New("user not found")

type setupOptions struct {
	userRepo    *mocks.MockUserRepository
	roleRepo    *mocks.MockRoleRepository
	sessionRepo *mocks.MockSessionRepository
	storage     *mocks.MockClient
	permEngine  services.PermissionEngine
	cfg         *config.Config
}

func newDefaultUserRepo(t *testing.T) *mocks.MockUserRepository {
	t.Helper()
	repo := mocks.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdatePassword", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	return repo
}

func newDefaultRoleRepo(t *testing.T) *mocks.MockRoleRepository {
	t.Helper()
	repo := mocks.NewMockRoleRepository(t)
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*permission.Role]{Items: []*permission.Role{}, Total: 0}, nil)
	repo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(&permission.Role{}, nil)
	repo.On("GetRolesWithInheritance", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetUsersWithRole", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetUserRoleAssignments", mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil, nil)
	repo.On("CreateAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("DeleteAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("CreateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("UpdateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("DeleteResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetResourcePermissionsByRoleID", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	return repo
}

func newDefaultSessionRepo(t *testing.T) *mocks.MockSessionRepository {
	t.Helper()
	repo := mocks.NewMockSessionRepository(t)
	repo.On("Get", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)
	return repo
}

func newDefaultPermCacheRepo(t *testing.T) *mocks.MockPermissionCacheRepository {
	t.Helper()
	repo := mocks.NewMockPermissionCacheRepository(t)
	repo.On("Get", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("Delete", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("InvalidateByRole", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("InvalidateOrganization", mock.Anything, mock.Anything).Maybe().Return(nil)
	return repo
}

func setupUserHandler(t *testing.T, opts setupOptions) *userhandler.Handler {
	t.Helper()

	if opts.userRepo == nil {
		opts.userRepo = newDefaultUserRepo(t)
	}
	if opts.roleRepo == nil {
		opts.roleRepo = newDefaultRoleRepo(t)
	}
	if opts.sessionRepo == nil {
		opts.sessionRepo = newDefaultSessionRepo(t)
	}
	if opts.permEngine == nil {
		opts.permEngine = &mocks.AllowAllPermissionEngine{}
	}
	if opts.storage == nil {
		opts.storage = mocks.NewMockClient(t)
		opts.storage.On("Upload", mock.Anything, mock.Anything).Maybe().Return((*storage.FileInfo)(nil), nil)
		opts.storage.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)
		opts.storage.On("GetPresignedURL", mock.Anything, mock.Anything).
			Maybe().
			Return("https://example.test/profile-picture.png", nil)
	}
	if opts.cfg == nil {
		opts.cfg = &config.Config{
			App: config.AppConfig{Debug: true},
			Security: config.SecurityConfig{
				Session: config.SessionConfig{
					Name: "trenova_session",
				},
			},
			Storage: config.StorageConfig{
				MaxFileSize:         5 * 1024 * 1024,
				PresignedURLExpiry:  15 * time.Minute,
				AllowedMIMETypes:    []string{"image/jpeg", "image/png", "image/webp"},
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
		Config:            opts.cfg,
		Validator:         userservice.NewTestValidator(),
	})

	roleSvc := roleservice.New(roleservice.Params{
		Logger:           logger,
		RoleRepo:         opts.roleRepo,
		UserRepo:         opts.userRepo,
		PermissionCache:  newDefaultPermCacheRepo(t),
		PermissionEngine: opts.permEngine,
		Validator:        roleservice.NewTestValidator(),
		Registry:         permission.NewEmptyRegistry(),
	})

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: opts.cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: opts.permEngine,
		ErrorHandler:     errorHandler,
	})

	return userhandler.New(userhandler.Params{
		Service:              userSvc,
		RoleService:          roleSvc,
		Config:               opts.cfg,
		PermissionEngine:     opts.permEngine,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	})
}

func TestUserHandler_List_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockUserRepository(t)
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
		Total: 1,
	}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdatePassword", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)

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

	repo := mocks.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*tenant.User]{
		Items: []*tenant.User{},
		Total: 50,
	}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdatePassword", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithQuery(map[string]string{"limit": "10", "offset": "20"}).
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp pagination.Response[[]map[string]any]
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, 50, resp.Count)
}

func TestUserHandler_List_Error(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockUserRepository(t)
	repo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("database error"))
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func makeUserRepo(t *testing.T) *mocks.MockUserRepository {
	t.Helper()
	repo := newDefaultUserRepo(t)
	return repo
}

func makeUserRepoWithGetByID(t *testing.T, user *tenant.User, err error) *mocks.MockUserRepository {
	t.Helper()
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(user, err)
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
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

func TestUserHandler_Update_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
		Timezone:              "America/New_York",
	}, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tenant.User) (*tenant.User, error) {
			return entity, nil
		})
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "Updated User",
			"username":     "updateduser",
			"emailAddress": "updated@example.com",
			"status":       "Active",
			"timezone":     "America/New_York",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Updated User", resp["name"])
}

func TestUserHandler_Update_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Updated User",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUserHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_Update_ServiceError(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
		Timezone:              "America/New_York",
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPut).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name":         "Updated User",
			"username":     "updateduser",
			"emailAddress": "updated@example.com",
			"status":       "Active",
			"timezone":     "America/New_York",
		})

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
	roleRepo := newDefaultRoleRepo(t)
	roleRepo.ExpectedCalls = nil
	roleRepo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*permission.Role]{Items: []*permission.Role{}, Total: 0}, nil)
	roleRepo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(&permission.Role{}, nil)
	roleRepo.On("GetRolesWithInheritance", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	roleRepo.On("GetUsersWithRole", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", mock.Anything, mock.Anything, mock.Anything).
		Return([]*permission.UserRoleAssignment{
			{
				ID:     pulid.MustNew("ura_"),
				UserID: userID,
				RoleID: pulid.MustNew("rol_"),
			},
		}, nil)
	roleRepo.On("CreateAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("DeleteAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("CreateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("UpdateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("DeleteResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("GetResourcePermissionsByRoleID", mock.Anything, mock.Anything).
		Maybe().
		Return(nil, nil)

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
	roleRepo := newDefaultRoleRepo(t)
	roleRepo.ExpectedCalls = nil
	roleRepo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*permission.Role]{Items: []*permission.Role{}, Total: 0}, nil)
	roleRepo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(&permission.Role{}, nil)
	roleRepo.On("GetRolesWithInheritance", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	roleRepo.On("GetUsersWithRole", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("role repo error"))
	roleRepo.On("CreateAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("DeleteAssignment", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("CreateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("UpdateResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("DeleteResourcePermission", mock.Anything, mock.Anything).Maybe().Return(nil)
	roleRepo.On("GetResourcePermissionsByRoleID", mock.Anything, mock.Anything).
		Maybe().
		Return(nil, nil)

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
	permEngine := mocks.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Return(&services.PermissionCheckResult{Allowed: true}, nil)
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

func TestUserHandler_Patch_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
		Timezone:              "America/New_York",
	}, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *tenant.User) (*tenant.User, error) {
			return entity, nil
		})
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched User",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())

	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Patched User", resp["name"])
}

func TestUserHandler_Patch_InvalidID(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/invalid-id/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched User",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusBadRequest, ginCtx.ResponseCode())
}

func TestUserHandler_Patch_GetByIDError(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := makeUserRepoWithGetByID(t, nil, errNotFound)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched User",
		})

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusInternalServerError, ginCtx.ResponseCode())
}

func TestUserHandler_Patch_BadJSON(t *testing.T) {
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
		Timezone:              "America/New_York",
	}, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithBody("{invalid json")

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.True(t, ginCtx.ResponseCode() >= 400)
}

func TestUserHandler_Patch_UpdateError(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(&tenant.User{
		ID:                    userID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Status:                domaintypes.StatusActive,
		Timezone:              "America/New_York",
	}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPatch).
		WithPath("/api/v1/users/" + userID.String() + "/").
		WithDefaultAuthContext().
		WithJSONBody(map[string]any{
			"name": "Patched User",
		})

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
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{userRepo: repo})

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
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

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
			Total: 1,
		}, nil)
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

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
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("GetOrganizations", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

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

	permEngine := mocks.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Maybe().
		Return(&services.PermissionCheckResult{Allowed: true}, nil)
	permEngine.On("SimulatePermissions", mock.Anything, mock.Anything).
		Return(&services.EffectivePermissions{
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
	permEngine := mocks.NewMockPermissionEngine(t)
	permEngine.On("Check", mock.Anything, mock.Anything).
		Maybe().
		Return(&services.PermissionCheckResult{Allowed: true}, nil)
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
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

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
	repo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	repo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	repo.On("UpdateCurrentOrganization", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	repo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	repo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	repo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

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

	sessionRepo := mocks.NewMockSessionRepository(t)
	sessionRepo.On("Get", mock.Anything, mock.Anything).Return(&session.Session{
		ID:             sessionID,
		UserID:         testutil.TestUserID,
		OrganizationID: testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
	}, nil)
	sessionRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	sessionRepo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)

	userRepo := mocks.NewMockUserRepository(t)
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
	userRepo.On("List", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	userRepo.On("SelectOptions", mock.Anything, mock.Anything).
		Maybe().
		Return(&pagination.ListResult[*tenant.User]{Items: []*tenant.User{}, Total: 0}, nil)
	userRepo.On("FindByEmail", mock.Anything, mock.Anything).Maybe().Return(nil, errNotFound)
	userRepo.On("UpdateLastLoginAt", mock.Anything, mock.Anything).Maybe().Return(nil)
	userRepo.On("IsPlatformAdmin", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	userRepo.On("GetUserOrganizationSummaries", mock.Anything, mock.Anything).
		Maybe().
		Return(nil, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("BulkUpdateStatus", mock.Anything, mock.Anything).Maybe().Return(nil, nil)
	userRepo.On("GetByIDs", mock.Anything, mock.Anything).Maybe().Return(nil, nil)

	handler := setupUserHandler(t, setupOptions{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultAuthContext().
		WithHeader("Cookie", "trenova_session="+sessionID.String()).
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

func TestUserHandler_SwitchOrganization_InvalidSessionID(t *testing.T) {
	t.Parallel()

	handler := setupUserHandler(t, setupOptions{})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultAuthContext().
		WithHeader("Cookie", "trenova_session=invalid-session-id").
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
		WithDefaultAuthContext().
		WithHeader("Cookie", "trenova_session="+sessionID.String()).
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
		WithDefaultAuthContext().
		WithHeader("Cookie", "trenova_session="+sessionID.String()).
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

	sessionRepo := mocks.NewMockSessionRepository(t)
	sessionRepo.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("session not found"))
	sessionRepo.On("Create", mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("Update", mock.Anything, mock.Anything).Maybe().Return(nil)
	sessionRepo.On("Delete", mock.Anything, mock.Anything).Maybe().Return(nil)

	handler := setupUserHandler(t, setupOptions{sessionRepo: sessionRepo})

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/users/me/switch-organization/").
		WithDefaultAuthContext().
		WithHeader("Cookie", "trenova_session="+sessionID.String()).
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

	repo := mocks.NewMockUserRepository(t)
	repo.On("GetByID", mock.Anything, mock.Anything).Return(newTestUserForHandler(), nil).Twice()
	repo.On("Update", mock.Anything, mock.Anything).Return(user, nil).Once()

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

func TestUserHandler_ChangeMyPassword_Success(t *testing.T) {
	t.Parallel()

	user := newTestUserForHandler()
	hashedPassword, err := user.GeneratePassword("current-password")
	require.NoError(t, err)
	user.Password = hashedPassword

	repo := mocks.NewMockUserRepository(t)
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
