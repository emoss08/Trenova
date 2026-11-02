package authservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	testTokens        = sessiontoken.NewIssuer("unit-test-session-secret-with-32-bytes")
	testSessionUserID = pulid.MustNew("usr_")
)

func storedSession(id pulid.ID, secretHash string) *session.Session {
	return &session.Session{
		ID:             id,
		SecretHash:     secretHash,
		UserID:         testSessionUserID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ExpiresAt:      time.Now().Add(time.Hour).Unix(),
		LastAccessedAt: time.Now().Unix(),
	}
}

func TestAuthenticateSession_RecordsWhenAQuietSessionIsSeenAgain(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.LastAccessedAt = time.Now().Add(-time.Hour).Unix()
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)
	deps.sessionRepo.On("Touch", mock.Anything, sessionID, mock.AnythingOfType("int64")).Return(nil)

	sess, err := deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(sess.LastAccessedAt, 0), 5*time.Second)
	deps.sessionRepo.AssertCalled(
		t,
		"Touch",
		mock.Anything,
		sessionID,
		mock.AnythingOfType("int64"),
	)
}

func TestAuthenticateSession_DoesNotWriteForEveryRequest(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.LastAccessedAt = time.Now().Add(-time.Minute).Unix()
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.NoError(t, err)
	deps.sessionRepo.AssertNotCalled(t, "Touch", mock.Anything, mock.Anything, mock.Anything)
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

	resp, err := deps.svc.Login(t.Context(), tenantports.LoginRequest{
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
	assert.True(t, testTokens.Matches(created.ID, created.UserID, secret, created.SecretHash))
	assert.NotContains(t, created.SecretHash, secret)
	assert.True(t, created.IssuedBySignIn)
}

func TestAuthenticateSession_AcceptsTheIssuedToken(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
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
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, issued.SecretHash), nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), sessionID.String()+".guessed-secret")

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	require.ErrorIs(t, err, sessionports.ErrSessionSecretMismatch)
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

func TestKeepSessionAlive_RecordsActivityEvenInsideTheTouchResolution(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.LastAccessedAt = time.Now().Add(-time.Minute).Unix()
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)
	deps.sessionRepo.On("Touch", mock.Anything, sessionID, mock.AnythingOfType("int64")).
		Return(nil).
		Once()

	sess, err := deps.svc.KeepSessionAlive(t.Context(), issued.Token)

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), time.Unix(sess.LastAccessedAt, 0), 5*time.Second)
	deps.sessionRepo.AssertNumberOfCalls(t, "Touch", 1)
}

func TestKeepSessionAlive_RefusesATokenThatDoesNotAuthenticate(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	other, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, other.SecretHash), nil)

	_, err = deps.svc.KeepSessionAlive(t.Context(), issued.Token)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	deps.sessionRepo.AssertNotCalled(t, "Touch", mock.Anything, mock.Anything, mock.Anything)
}

func TestKeepSessionAlive_FailsWhenActivityCannotBeRecorded(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, issued.SecretHash)
	stored.LastAccessedAt = time.Now().Add(-time.Minute).Unix()
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)
	deps.sessionRepo.On("Touch", mock.Anything, sessionID, mock.AnythingOfType("int64")).
		Return(errors.New("redis unavailable"))

	_, err = deps.svc.KeepSessionAlive(t.Context(), issued.Token)

	require.Error(t, err)
}

func TestAuthenticateSession_AnEndedSessionIsAnAuthenticationError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(nil, sessionports.ErrSessionNotFound)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
}

func TestAuthenticateSession_AStoreFailureDoesNotEndTheSession(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	outage := errors.New("redis: connection refused")
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(nil, outage)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.ErrorIs(t, err, outage)
	assert.False(t, errortypes.IsAuthenticationError(err))
	deps.sessionRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestAuthenticateSession_AcceptsThePreviousSecretDuringTheGrace(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	previous, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	current, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, previous.SecretHash)
	stored.ReplaceSecret(current.SecretHash, time.Now().Unix(), session.RotationGrace)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	sess, err := deps.svc.AuthenticateSession(t.Context(), previous.Token)

	require.NoError(t, err)
	assert.Equal(t, previous.SecretHash, sess.AuthenticatedSecretHash)
}

func TestAuthenticateSession_RefusesThePreviousSecretAfterTheGrace(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	previous, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	current, err := testTokens.Issue(sessionID, testSessionUserID)
	require.NoError(t, err)
	stored := storedSession(sessionID, previous.SecretHash)
	stored.ReplaceSecret(
		current.SecretHash,
		time.Now().Add(-2*session.RotationGrace).Unix(),
		session.RotationGrace,
	)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).Return(stored, nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), previous.Token)

	require.ErrorIs(t, err, sessionports.ErrSessionSecretMismatch)
}

func TestAuthenticateSession_RefusesASecretIssuedForAnotherUser(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	sessionID := pulid.MustNew("ses_")
	issued, err := testTokens.Issue(sessionID, pulid.MustNew("usr_"))
	require.NoError(t, err)
	deps.sessionRepo.On("Get", mock.Anything, sessionID).
		Return(storedSession(sessionID, issued.SecretHash), nil)

	_, err = deps.svc.AuthenticateSession(t.Context(), issued.Token)

	require.ErrorIs(t, err, sessionports.ErrSessionSecretMismatch)
}

func TestLogin_DiscardsASessionWhoseCredentialsChangedWhileItSignedIn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	deps.userRepo.ExpectedCalls = nil
	deps.userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(usr, nil)
	deps.userRepo.On("GetSessionEpoch", mock.Anything, usr.ID).
		Return(usr.SessionEpoch+1, nil).
		Once()
	deps.portalRepo.On("ExistsWorkerForUser", mock.Anything, mock.Anything).Return(false, nil)
	var created *session.Session
	deps.sessionRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(1).(*session.Session) }).
		Return(nil).
		Once()
	deps.sessionRepo.On("Delete", mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := deps.svc.Login(t.Context(), tenantports.LoginRequest{
		EmailAddress: "test@example.com",
		Password:     "password123",
	})

	require.ErrorIs(t, err, errInvalidCredentials)
	assert.Nil(t, resp)
	require.NotNil(t, created)
	deps.sessionRepo.AssertCalled(t, "Delete", mock.Anything, created.ID)
}
