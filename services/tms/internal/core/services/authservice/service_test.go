package authservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
		ur: ur,
		sr: sr,
		l:  zap.NewNop(),
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
