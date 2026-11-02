//go:build integration

package authstore

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
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
		SecretHash:     "secret-hash",
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CreatedAt:      createdAt,
		LastAccessedAt: lastAccessedAt,
		ExpiresAt:      expiresAt,
	}
}

// seedSession stores a session the way Create does but without its check, so a test can
// hold one that has already ended, as a session stored before the policy changed would be.
func seedSession(t *testing.T, repo *sessionRepository, sess *session.Session) {
	t.Helper()

	require.NoError(t, repo.seal(sess))

	require.NoError(t, redishelpers.SetJSON(
		t.Context(),
		repo.client,
		sessionKey(sess.ID),
		sess,
		time.Hour,
	))
	require.NoError(t, repo.client.ZAdd(t.Context(), repo.getUserSessionsKey(sess.UserID), redis.Z{
		Score:  float64(sess.CreatedAt),
		Member: sess.ID.String(),
	}).Err())
}

func TestSessionRepository_KeyLivesOnlyAsLongAsTheSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	expiresAt := integrationSessionPolicy.ExpiresAt(now)
	sess := newIntegrationSession(pulid.MustNew("usr_"), now, now, expiresAt)

	require.NoError(t, repo.Create(t.Context(), sess))

	ttl, err := repo.client.TTL(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.InDelta(t, (12 * time.Hour).Seconds(), ttl.Seconds(), 5)
}

func TestSessionRepository_GetEndsAnIdleSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	idleFor := integrationIdleFor
	sess := newIntegrationSession(userID, now-idleFor, now-idleFor, now+60*60)
	seedSession(t, repo, sess)

	_, err := repo.Get(t.Context(), sess.ID)
	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)

	exists, err := repo.client.Exists(t.Context(), sessionKey(sess.ID)).Result()
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

func TestSessionRepository_CreateRefusesASessionThatHasAlreadyEnded_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	idleFor := integrationIdleFor
	sess := newIntegrationSession(pulid.MustNew("usr_"), now-idleFor, now-idleFor, now+60*60)

	require.Error(t, repo.Create(t.Context(), sess))

	exists, err := repo.client.Exists(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
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
	seedSession(t, repo, idle)

	sessions, err := repo.ListForUser(t.Context(), userID)

	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, active.ID, sessions[0].ID)
	members, err := repo.client.ZRange(t.Context(), repo.getUserSessionsKey(userID), 0, -1).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{active.ID.String()}, members)
}

func endedIDs(standings []sessionports.SessionStanding) []pulid.ID {
	ended := make([]pulid.ID, 0)
	for _, standing := range standings {
		if standing.Ended {
			ended = append(ended, standing.ID)
		}
	}
	return ended
}

func TestSessionRepository_StandingsNameEverySessionThatNoLongerHolds_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	live := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	idle := newIntegrationSession(
		userID,
		now-integrationIdleFor,
		now-integrationIdleFor,
		now+60*60,
	)
	past := newIntegrationSession(userID, now-60, now-60, now-1)
	deleted := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	seedSession(t, repo, live)
	seedSession(t, repo, idle)
	seedSession(t, repo, past)
	require.NoError(t, repo.Create(t.Context(), deleted))
	require.NoError(t, repo.Delete(t.Context(), deleted.ID))
	never := pulid.MustNew("ses_")

	standings, err := repo.Standings(
		t.Context(),
		[]pulid.ID{live.ID, idle.ID, past.ID, deleted.ID, never},
	)

	require.NoError(t, err)
	assert.ElementsMatch(t, []pulid.ID{idle.ID, past.ID, deleted.ID, never}, endedIDs(standings))
}

func TestSessionRepository_StandingsReportTheSecretsASessionAccepts_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	graced := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	graced.ReplaceSecret("new-secret-hash", now, session.RotationGrace)
	lapsed := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	lapsed.ReplaceSecret(
		"new-secret-hash",
		now-int64(2*session.RotationGrace/time.Second),
		session.RotationGrace,
	)
	seedSession(t, repo, graced)
	seedSession(t, repo, lapsed)

	standings, err := repo.Standings(t.Context(), []pulid.ID{graced.ID, lapsed.ID})

	require.NoError(t, err)
	require.Len(t, standings, 2)
	assert.True(t, standings[0].SecretsKnown)
	assert.Equal(t, []string{"new-secret-hash", "secret-hash"}, standings[0].SecretHashes)
	assert.True(t, standings[1].SecretsKnown)
	assert.Equal(t, []string{"new-secret-hash"}, standings[1].SecretHashes)
}

func TestSessionRepository_StandingsCheckSessionsInBatches_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	ids := make([]pulid.ID, 0, livenessBatchSize+3)
	for range livenessBatchSize + 3 {
		ids = append(ids, pulid.MustNew("ses_"))
	}
	live := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	seedSession(t, repo, live)
	ids = append(ids, live.ID)

	standings, err := repo.Standings(t.Context(), ids)

	require.NoError(t, err)
	ended := endedIDs(standings)
	assert.Len(t, ended, livenessBatchSize+3)
	assert.NotContains(t, ended, live.ID)
}

func TestSessionRepository_UpdateNeverRecreatesASessionDeletedMidUpdate_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	sess := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	sess.SecretHash = "secret-hash"
	require.NoError(t, repo.Create(t.Context(), sess))

	calls := 0
	_, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		calls++
		require.NoError(t, repo.DeleteAllForUser(t.Context(), userID))
		current.Locale = "es"
		return nil
	})

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
	assert.Equal(t, 1, calls)
	exists, err := repo.client.Exists(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}

func TestSessionRepository_UpdateReappliesAChangeOverAConcurrentWrite_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	sess := newIntegrationSession(
		pulid.MustNew("usr_"),
		now,
		now,
		integrationSessionPolicy.ExpiresAt(now),
	)
	sess.SecretHash = "secret-hash"
	require.NoError(t, repo.Create(t.Context(), sess))

	calls := 0
	updated, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		calls++
		if calls == 1 {
			require.NoError(t, repo.client.JSONSet(
				t.Context(),
				sessionKey(sess.ID),
				".locale",
				`"fr"`,
			).Err())
		}
		current.ActiveRoleIDs = []pulid.ID{pulid.MustNew("rol_")}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	got, err := repo.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "fr", got.Locale)
	assert.Equal(t, updated.ActiveRoleIDs, got.ActiveRoleIDs)
	assert.Equal(t, "secret-hash", got.SecretHash)
	ttl, err := repo.client.TTL(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.InDelta(t, (12 * time.Hour).Seconds(), ttl.Seconds(), 5)
}

func TestSessionRepository_UpdateReportsAMissingSessionAsNotFound_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	sessionID := pulid.MustNew("ses_")

	_, err := repo.Update(t.Context(), sessionID, func(*session.Session) error {
		return nil
	})

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
	exists, err := repo.client.Exists(t.Context(), sessionKey(sessionID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}

func TestSessionRepository_UpdateLeavesTheSessionAloneWhenTheChangeFails_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	sess := newIntegrationSession(
		pulid.MustNew("usr_"),
		now,
		now,
		integrationSessionPolicy.ExpiresAt(now),
	)
	sess.Locale = "en"
	require.NoError(t, repo.Create(t.Context(), sess))
	errRefused := errors.New("refused")

	_, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		current.Locale = "es"
		return errRefused
	})

	require.ErrorIs(t, err, errRefused)
	got, err := repo.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "en", got.Locale)
}

func TestSessionRepository_DeleteOfAMissingSessionSucceeds_Integration(t *testing.T) {
	repo := setupSessionRepository(t)

	require.NoError(t, repo.Delete(t.Context(), pulid.MustNew("ses_")))
}

func TestSessionRepository_UpdateEndsASessionThePolicyHasEnded_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	idleFor := integrationIdleFor
	sess := newIntegrationSession(userID, now-idleFor, now-idleFor, now+60*60)
	seedSession(t, repo, sess)

	_, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		current.Locale = "es"
		return nil
	})

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
	exists, err := repo.client.Exists(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
	members, err := repo.client.ZRange(t.Context(), repo.getUserSessionsKey(userID), 0, -1).Result()
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestSessionRepository_UpdateGivesUpOnASessionThatKeepsChanging_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	sess := newIntegrationSession(
		pulid.MustNew("usr_"),
		now,
		now,
		integrationSessionPolicy.ExpiresAt(now),
	)
	sess.Locale = "en"
	require.NoError(t, repo.Create(t.Context(), sess))

	calls := 0
	_, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		calls++
		require.NoError(t, repo.Touch(t.Context(), sess.ID, now+int64(calls)))
		current.Locale = "es"
		return nil
	})

	require.ErrorIs(t, err, redis.TxFailedErr)
	assert.Equal(t, maxUpdateAttempts, calls)
	got, err := repo.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, "en", got.Locale)
}

func TestSessionRepository_ConfirmAccessRecordsActivityAndConfirmation_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	sess := newIntegrationSession(pulid.MustNew("usr_"), now-600, now-600, now+60*60)
	seedSession(t, repo, sess)

	require.NoError(t, repo.ConfirmAccess(t.Context(), sess.ID, now))

	got, err := repo.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, now, got.LastAccessedAt)
	assert.Equal(t, now, got.AccessConfirmedAt)
}

func TestSessionRepository_ConfirmAccessNeverRecreatesAMissingSession_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	sessionID := pulid.MustNew("ses_")

	_ = repo.ConfirmAccess(t.Context(), sessionID, timeutils.NowUnix())

	exists, err := repo.client.Exists(t.Context(), sessionKey(sessionID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}

func TestSessionRepository_GetDiscardsASessionEditedOutsideTheServer_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	sess := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	require.NoError(t, repo.Create(t.Context(), sess))
	require.NoError(t, repo.client.JSONSet(
		t.Context(),
		sessionKey(sess.ID),
		".organizationId",
		`"`+pulid.MustNew("org_").String()+`"`,
	).Err())

	_, err := repo.Get(t.Context(), sess.ID)

	require.ErrorIs(t, err, sessionports.ErrSessionNotFound)
	exists, err := repo.client.Exists(t.Context(), sessionKey(sess.ID)).Result()
	require.NoError(t, err)
	assert.Zero(t, exists)
}

func TestSessionRepository_UpdateRefusesToChangeWhoTheSessionIs_Integration(t *testing.T) {
	repo := setupSessionRepository(t)
	now := timeutils.NowUnix()
	userID := pulid.MustNew("usr_")
	sess := newIntegrationSession(userID, now, now, integrationSessionPolicy.ExpiresAt(now))
	require.NoError(t, repo.Create(t.Context(), sess))

	_, err := repo.Update(t.Context(), sess.ID, func(current *session.Session) error {
		current.UserID = pulid.MustNew("usr_")
		return nil
	})

	require.ErrorIs(t, err, errIdentityChanged)
	got, err := repo.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
}
