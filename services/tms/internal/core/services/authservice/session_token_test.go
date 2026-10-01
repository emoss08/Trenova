package authservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func storedSession(id pulid.ID, secretHash string) *session.Session {
	return &session.Session{
		ID:             id,
		SecretHash:     secretHash,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
	}
}

func TestLogin_IssuesASessionTokenWhoseSecretIsStoredOnlyAsAHash(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)

	var created *session.Session
	deps.userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(usr, nil)
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(1).(*session.Session) }).
		Return(nil)
	deps.userRepo.On("UpdateLastLoginAt", mock.Anything, usr.ID).Return(nil)

	resp, err := deps.svc.Login(t.Context(), services.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	})
	require.NoError(t, err)
	require.NotNil(t, created)

	sessionID, secret, err := sessiontoken.Parse(resp.SessionToken)
	require.NoError(t, err)
	assert.Equal(t, created.ID, sessionID)
	assert.Equal(t, created.ID.String(), resp.SessionID)
	assert.NotContains(t, resp.SessionID, secret)
	assert.True(t, sessiontoken.Matches(secret, created.SecretHash))
	assert.NotContains(t, created.SecretHash, secret)
}

func TestAuthenticateSession_AcceptsTheIssuedToken(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, issued.SecretHash), nil)

	sess, err := deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.NoError(t, err)
	assert.Equal(t, sessionID, sess.ID)
}

func TestAuthenticateSession_RejectsAGuessedSecretWithoutDeletingTheSession(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := sessiontoken.Issue(sessionID)
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, issued.SecretHash), nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), sessionID.String()+".guessed-secret")

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestAuthenticateSession_RejectsTheBareSessionIDCookieOfOlderReleases(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	_, err := deps.svc.AuthenticateSession(t.Context(), pulid.MustNew("ses_").String())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	deps.sessionRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestAuthenticateSession_RejectsASessionStoredWithoutASecret(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, ""), nil)

	_, err := deps.svc.AuthenticateSession(t.Context(), sessionID.String()+".any-secret")

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
}
