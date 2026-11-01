package userhandler_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/userhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type setupOptions struct {
	userRepo    *tenantportsmock.MockUserRepository
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

	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    logger,
		AppConfig: &opts.cfg.App,
	})

	return userhandler.New(userhandler.Params{
		Service:      userSvc,
		ErrorHandler: errorHandler,
	})
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
