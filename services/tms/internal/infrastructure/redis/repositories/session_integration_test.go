//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var integrationSessionPolicy = session.Policy{
	MaxAge:      12 * time.Hour,
	IdleTimeout: 30 * time.Minute,
}

var integrationIdleFor = int64(
	(integrationSessionPolicy.IdleTimeout + session.IdleGrace + time.Minute) / time.Second,
)

func setupSessionRepository(t *testing.T) *sessionRepository {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	t.Cleanup(func() { client.FlushDB(t.Context()) })

	return &sessionRepository{
		client: client,
		policy: integrationSessionPolicy,
		l:      zap.NewNop(),
	}
}

func newIntegrationSession(
	userID pulid.ID,
	createdAt, lastAccessedAt, expiresAt int64,
) *session.Session {
	return &session.Session{
		ID:             pulid.MustNew("ses_"),
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CreatedAt:      createdAt,
		LastAccessedAt: lastAccessedAt,
		ExpiresAt:      expiresAt,
	}
}

func TestSessionRepository_KeyLivesOnlyAsLongAsTheSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	expiresAt := integrationSessionPolicy.ExpiresAt(now)
	sess := newIntegrationSession(pulid.MustNew("usr_"), now, now, expiresAt)

	require.NoError(t, repo.Create(t.Context(), sess))

	ttl, err := repo.client.TTL(t.Context(), repo.getSessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.InDelta(t, (12 * time.Hour).Seconds(), ttl.Seconds(), 5)
}

func TestSessionRepository_GetEndsAnIdleSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	idleFor := integrationIdleFor
	sess := newIntegrationSession(userID, now-idleFor, now-idleFor, now+60*60)
	require.NoError(t, repo.Create(t.Context(), sess))

	_, err := repo.Get(t.Context(), sess.ID)
	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)

	exists, err := repo.client.Exists(t.Context(), repo.getSessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
	members, err := repo.client.ZRange(t.Context(), repo.getUserSessionsKey(userID), 0, -1).Result()
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestSessionRepository_GetCapsALongerLivedSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	createdAt := now - 60*60
	sess := newIntegrationSession(pulid.MustNew("usr_"), createdAt, now, createdAt+30*24*60*60)
	require.NoError(t, repo.Create(t.Context(), sess))

	got, err := repo.Get(t.Context(), sess.ID)

	require.NoError(t, err)
	assert.Equal(t, integrationSessionPolicy.ExpiresAt(createdAt), got.ExpiresAt)
}

func TestSessionRepository_GetReportsAMissingSessionAsNotFound_Integration(t *testing.T) {
	repo := setupSessionRepository(t)

	_, err := repo.Get(t.Context(), pulid.MustNew("ses_"))

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
}

func TestSessionRepository_ListForUserLeavesOutEndedSessions_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	idleFor := integrationIdleFor

	activeExpiresAt := integrationSessionPolicy.ExpiresAt(now - 60)
	active := newIntegrationSession(userID, now-60, now-60, activeExpiresAt)
	idle := newIntegrationSession(userID, now-idleFor, now-idleFor, now+60*60)
	require.NoError(t, repo.Create(t.Context(), active))
	require.NoError(t, repo.Create(t.Context(), idle))

	sessions, err := repo.ListForUser(t.Context(), userID)

	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, active.ID, sessions[0].ID)
	members, err := repo.client.ZRange(t.Context(), repo.getUserSessionsKey(userID), 0, -1).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{active.ID.String()}, members)
}
