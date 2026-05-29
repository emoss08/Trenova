package authservice

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/rbactest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testDeps struct {
	userRepo    *mocks.MockUserRepository
	sessionRepo *mocks.MockSessionRepository
	svc         *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()
	ur := mocks.NewMockUserRepository(t)
	sr := mocks.NewMockSessionRepository(t)
	svc := &Service{
		ur:       ur,
		sr:       sr,
		rbacRepo: &rbactest.Repository{},
		l:        zap.NewNop(),
	}
	return &testDeps{userRepo: ur, sessionRepo: sr, svc: svc}
}

func newTestUser(t *testing.T) *tenant.User {
	t.Helper()
	usr := &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "Test User",
		Username:              "testuser",
		EmailAddress:          "test@example.com",
		Timezone:              "America/New_York",
	}
	hashed, err := usr.GeneratePassword("password123")
	require.NoError(t, err)
	usr.Password = hashed
	return usr
}

func TestDecryptSSOSecretSupportsIAMSlugAAD(t *testing.T) {
	t.Parallel()

	enc := encryptionservice.New(encryptionservice.Params{
		Config: &config.Config{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Key: "unit-test-encryption-key-with-at-least-32-bytes",
				},
			},
		},
	})
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	ciphertext, err := enc.EncryptStringWithAAD(
		"client-secret",
		encryptionservice.AAD{
			Purpose:        encryptionservice.PurposeIAMOIDCClientSecret,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ResourceID:     "azuread",
		},
	)
	require.NoError(t, err)

	plaintext := decryptSSOSecret(enc, &tenant.SSOConfig{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		Provider:         tenant.SSOProviderAzureAD,
		OIDCClientSecret: ciphertext,
	})

	require.Equal(t, "client-secret", plaintext)
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	result, err := deps.svc.Login(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, usr, result.User)
	assert.NotEmpty(t, result.SessionID)
	assert.Greater(t, result.ExpiresAt, timeutils.NowUnix())
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogin_IncludesAuthorizedRoleSummaries(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)
	roleID := pulid.MustNew("rol_")
	deps.svc.rbacRepo = &rbactest.Repository{
		AuthorizedRoles: []*permission.Role{
			{
				ID:          roleID,
				Name:        "Dispatcher",
				Description: "Coordinates dispatch activity",
			},
		},
	}

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	result, err := deps.svc.Login(ctx, req)

	require.NoError(t, err)
	require.Len(t, result.AuthorizedRoleIDs, 1)
	require.Len(t, result.AuthorizedRoles, 1)
	assert.Equal(t, roleID, result.AuthorizedRoleIDs[0])
	assert.Equal(t, "Dispatcher", result.AuthorizedRoles[0].Name)
	assert.True(t, result.RequiresRoleActivation)
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogin_ValidationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := services.LoginRequest{
		EmailAddress: "",
		Password:     "",
	}

	result, err := deps.svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.userRepo.AssertNotCalled(t, "FindByEmail")
	deps.sessionRepo.AssertNotCalled(t, "Create")
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()

	req := services.LoginRequest{
		EmailAddress: "notfound@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).
		Return(nil, errors.New("user not found"))

	result, err := deps.svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errInvalidCredentials, err)
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertNotCalled(t, "Create")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "wrongpassword",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)

	result, err := deps.svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertNotCalled(t, "Create")
}

func TestLogin_SessionCreateError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("redis error"))

	result, err := deps.svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "redis error")
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertExpectations(t)
}

func TestValidateSession_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	sess := &session.Session{
		ID:        sessionID,
		ExpiresAt: timeutils.NowUnix() + 3600,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)

	result, err := deps.svc.ValidateSession(ctx, sessionID)

	require.NoError(t, err)
	assert.Equal(t, sess, result)
	deps.sessionRepo.AssertExpectations(t)
}

func TestActivateSessionRoles_IncludesRoleSummaries(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	deps.svc.rbacRepo = &rbactest.Repository{
		AuthorizedRoles: []*permission.Role{
			{
				ID:          roleID,
				Name:        "Billing Manager",
				Description: "Manages billing operations",
			},
		},
	}
	sess := &session.Session{
		ID:             sessionID,
		UserID:         userID,
		OrganizationID: orgID,
		ActiveRoleIDs:  []pulid.ID{},
		ExpiresAt:      timeutils.NowUnix() + 3600,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)
	deps.sessionRepo.On("Update", mock.Anything, mock.MatchedBy(func(updated *session.Session) bool {
		return len(updated.ActiveRoleIDs) == 1 && updated.ActiveRoleIDs[0] == roleID
	})).
		Return(nil)

	result, err := deps.svc.ActivateSessionRoles(ctx, services.ActivateSessionRolesRequest{
		SessionID: sessionID,
		RoleIDs:   []pulid.ID{roleID},
	})

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{roleID}, result.ActiveRoleIDs)
	assert.Equal(t, []pulid.ID{roleID}, result.AuthorizedRoleIDs)
	require.Len(t, result.ActiveRoles, 1)
	require.Len(t, result.AuthorizedRoles, 1)
	assert.Equal(t, "Billing Manager", result.ActiveRoles[0].Name)
	assert.False(t, result.RequiresRoleActivation)
	deps.sessionRepo.AssertExpectations(t)
}

func TestActivateSessionRoles_RejectsUnauthorizedRoleIDs(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	authorizedRoleID := pulid.MustNew("rol_")
	unauthorizedRoleID := pulid.MustNew("rol_")
	deps.svc.rbacRepo = &rbactest.Repository{
		AuthorizedRoles: []*permission.Role{
			{ID: authorizedRoleID, Name: "Dispatcher"},
		},
	}
	sess := &session.Session{
		ID:             sessionID,
		UserID:         userID,
		OrganizationID: orgID,
		ActiveRoleIDs:  []pulid.ID{},
		ExpiresAt:      timeutils.NowUnix() + 3600,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)

	result, err := deps.svc.ActivateSessionRoles(ctx, services.ActivateSessionRolesRequest{
		SessionID: sessionID,
		RoleIDs:   []pulid.ID{unauthorizedRoleID},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	deps.sessionRepo.AssertExpectations(t)
	deps.sessionRepo.AssertNotCalled(t, "Update")
}

func TestValidateSession_NotFound(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(nil, errors.New("session not found"))

	result, err := deps.svc.ValidateSession(ctx, sessionID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "session not found")
	deps.sessionRepo.AssertExpectations(t)
}

func TestValidateSession_Expired(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	sess := &session.Session{
		ID:        sessionID,
		ExpiresAt: timeutils.NowUnix() - 1000,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sessionID).Return(nil)

	result, err := deps.svc.ValidateSession(ctx, sessionID)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	deps.sessionRepo.On("Delete", mock.Anything, sessionID).Return(nil)

	err := deps.svc.Logout(ctx, sessionID)

	require.NoError(t, err)
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogout_Error(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	deps.sessionRepo.On("Delete", mock.Anything, sessionID).Return(errors.New("delete failed"))

	err := deps.svc.Logout(ctx, sessionID)

	require.Error(t, err)
	assert.EqualError(t, err, "delete failed")
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogin_UpdateLastLoginAtError_StillSucceeds(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(errors.New("update failed"))

	result, err := deps.svc.Login(ctx, req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, usr, result.User)
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertExpectations(t)
}

func TestStartSSOLogin_StoresProviderIDInState(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	providerID := pulid.MustNew("sso_")
	providerServer := newOIDCDiscoveryServer(t)

	orgRepo := mocks.NewMockOrganizationRepository(t)
	ssoRepo := mocks.NewMockSSOConfigRepository(t)
	stateRepo := mocks.NewMockSSOLoginStateRepository(t)
	svc := &Service{
		or:        orgRepo,
		ssoRepo:   ssoRepo,
		stateRepo: stateRepo,
		cfg: &config.Config{
			Server: config.ServerConfig{
				CORS: config.CORSConfig{
					AllowedOrigins: []string{"https://app.test"},
				},
			},
		},
		l: zap.NewNop(),
	}
	ssoConfig := &tenant.SSOConfig{
		ID:               providerID,
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		Name:             "Corporate SSO",
		Provider:         tenant.SSOProviderOkta,
		Protocol:         tenant.SSOProtocolOIDC,
		Enabled:          true,
		OIDCIssuerURL:    providerServer.URL,
		OIDCClientID:     "client-id",
		OIDCClientSecret: "client-secret",
		OIDCRedirectURL:  "https://api.test/auth/callback",
		OIDCScopes:       []string{"openid", "email", "profile"},
	}

	orgRepo.On("GetByLoginSlug", ctx, "acme").Return(&tenant.Organization{
		ID:             orgID,
		BusinessUnitID: buID,
		Name:           "Acme Logistics",
		LoginSlug:      "acme",
	}, nil)
	ssoRepo.On("GetEnabledByID", ctx, providerID).Return(ssoConfig, nil)
	stateRepo.On(
		"Save",
		ctx,
		mock.MatchedBy(func(state *repositories.SSOLoginState) bool {
			return state.ProviderID == providerID &&
				state.Provider == tenant.SSOProviderOkta &&
				state.OrganizationID == orgID &&
				state.OrganizationSlug == "acme" &&
				state.CodeVerifier != "" &&
				state.Nonce != "" &&
				state.ReturnTo == "https://app.test/home"
		}),
		ssoLoginStateTTL,
	).Return(nil)

	redirectURL, err := svc.StartSSOLogin(ctx, services.StartSSOLoginRequest{
		ProviderID:       providerID,
		OrganizationSlug: "acme",
		ReturnTo:         "https://app.test/home",
	})

	require.NoError(t, err)
	assert.Contains(t, redirectURL, providerServer.URL+"/authorize")
	orgRepo.AssertExpectations(t)
	ssoRepo.AssertExpectations(t)
	stateRepo.AssertExpectations(t)
}

func TestResolveSSOConfigForCallback_UsesProviderID(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	providerID := pulid.MustNew("sso_")
	ssoRepo := mocks.NewMockSSOConfigRepository(t)
	svc := &Service{ssoRepo: ssoRepo}
	expectedConfig := &tenant.SSOConfig{
		ID:             providerID,
		OrganizationID: orgID,
		Provider:       tenant.SSOProviderOkta,
	}

	ssoRepo.On("GetEnabledByID", ctx, providerID).Return(expectedConfig, nil)

	cfg, err := svc.resolveSSOConfigForCallback(ctx, &repositories.SSOLoginState{
		Provider:       tenant.SSOProviderAzureAD,
		ProviderID:     providerID,
		OrganizationID: orgID,
	})

	require.NoError(t, err)
	assert.Equal(t, expectedConfig, cfg)
	ssoRepo.AssertExpectations(t)
	ssoRepo.AssertNotCalled(t, "GetEnabledByOrganizationID")
}

func TestValidateSession_ExpiredDeleteError_StillReturnsExpiredErr(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	sessionID := pulid.MustNew("ses_")

	sess := &session.Session{
		ID:        sessionID,
		ExpiresAt: timeutils.NowUnix() - 1000,
	}

	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(sess, nil)
	deps.sessionRepo.On("Delete", mock.Anything, sessionID).Return(errors.New("delete failed"))

	result, err := deps.svc.ValidateSession(ctx, sessionID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "expired")
	deps.sessionRepo.AssertExpectations(t)
}

func TestLogin_InactiveUser(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	ctx := t.Context()
	usr := newTestUser(t)
	usr.Status = domaintypes.StatusInactive

	req := services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	}

	deps.userRepo.On("FindByEmail", mock.Anything, req.EmailAddress).Return(usr, nil)

	result, err := deps.svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	deps.userRepo.AssertExpectations(t)
	deps.sessionRepo.AssertNotCalled(t, "Create")
}

func TestNew(t *testing.T) {
	t.Parallel()

	userRepo := mocks.NewMockUserRepository(t)
	sessionRepo := mocks.NewMockSessionRepository(t)

	svc := New(Params{
		UserRepository:    userRepo,
		SessionRepository: sessionRepo,
		Logger:            zap.NewNop(),
	})

	require.NotNil(t, svc)
}

func newOIDCDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
			"issuer": "` + server.URL + `",
			"authorization_endpoint": "` + server.URL + `/authorize",
			"token_endpoint": "` + server.URL + `/token",
			"jwks_uri": "` + server.URL + `/keys",
			"response_types_supported": ["code"],
			"subject_types_supported": ["public"],
			"id_token_signing_alg_values_supported": ["RS256"]
		}`))
		require.NoError(t, err)
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	return server
}
