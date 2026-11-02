package sessionrotation_test

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/sessionrotation"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/sessionportsmock"
	"github.com/emoss08/trenova/internal/testutil/sessiontest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/sessiontoken"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setup(
	t *testing.T,
) (*sessionrotation.Rotator, *sessionportsmock.MockSessionRepository, *sessiontoken.Issuer) {
	t.Helper()

	repo := sessionportsmock.NewMockSessionRepository(t)
	tokens := sessiontoken.NewIssuer("unit-test-session-secret-with-32-bytes")
	return sessionrotation.New(sessionrotation.Params{Sessions: repo, Tokens: tokens}), repo, tokens
}

func storedSession() *session.Session {
	return &session.Session{
		ID:         pulid.MustNew("ses_"),
		UserID:     pulid.MustNew("usr_"),
		SecretHash: "old-secret-hash",
		ExpiresAt:  timeutils.NowUnix() + 3600,
	}
}

func TestRotateIssuesASecretOnlyTheNewTokenMatches(t *testing.T) {
	t.Parallel()

	rotator, repo, tokens := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()

	rotated, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: stored.SecretHash,
	})

	require.NoError(t, err)
	sessionID, secret, err := sessiontoken.Parse(rotated.Token)
	require.NoError(t, err)
	assert.Equal(t, stored.ID, sessionID)
	assert.True(t, tokens.Matches(stored.ID, stored.UserID, secret, stored.SecretHash))
	assert.Empty(t, stored.PreviousSecretHash)
}

func TestRotateCanKeepThePreviousSecretForTheGrace(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: stored.SecretHash,
		Grace:             session.RotationGrace,
	})

	require.NoError(t, err)
	assert.Equal(t, "old-secret-hash", stored.PreviousSecretHash)
	assert.Contains(t, stored.SecretHashesAt(timeutils.NowUnix()), "old-secret-hash")
}

func TestRotateAppliesTheChangeWithTheNewSecret(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: stored.SecretHash,
		Change: func(sess *session.Session) error {
			sess.Locale = "es"
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "es", stored.Locale)
	assert.NotEqual(t, "old-secret-hash", stored.SecretHash)
}

func TestRotateLeavesTheSessionAloneWhenTheChangeFails(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()
	refused := errors.New("refused")

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: stored.SecretHash,
		Change:            func(*session.Session) error { return refused },
	})

	require.ErrorIs(t, err, refused)
	assert.Equal(t, "old-secret-hash", stored.SecretHash)
}

func TestRotateRefusesAnotherUsersSession(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            pulid.MustNew("usr_"),
		CurrentSecretHash: stored.SecretHash,
	})

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
	assert.Equal(t, "old-secret-hash", stored.SecretHash)
}

func TestRotateRefusesAHolderOfASupersededSecret(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	stored.ReplaceSecret("current-secret-hash", timeutils.NowUnix(), session.RotationGrace)
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: "old-secret-hash",
		Grace:             session.RotationGrace,
	})

	assert.True(t, errortypes.IsSessionContextStaleError(err))
	assert.Equal(t, "current-secret-hash", stored.SecretHash)
	assert.Equal(t, "old-secret-hash", stored.PreviousSecretHash)
}

func TestRotateGivesACredentialChangeOnlyAShortGrace(t *testing.T) {
	t.Parallel()

	rotator, repo, _ := setup(t)
	stored := storedSession()
	repo.EXPECT().Update(mock.Anything, stored.ID, mock.Anything).
		RunAndReturn(sessiontest.UpdateStored(stored)).
		Once()
	now := timeutils.NowUnix()

	_, err := rotator.Rotate(t.Context(), sessionrotation.Request{
		SessionID:         stored.ID,
		UserID:            stored.UserID,
		CurrentSecretHash: stored.SecretHash,
		Grace:             session.CredentialChangeGrace,
	})

	require.NoError(t, err)
	grace := int64(session.CredentialChangeGrace / time.Second)
	assert.InDelta(t, now+grace, stored.PreviousSecretUntil, 2)
	assert.NotContains(t, stored.SecretHashesAt(now+grace+1), "old-secret-hash")
}
