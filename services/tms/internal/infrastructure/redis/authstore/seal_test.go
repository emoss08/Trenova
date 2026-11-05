package authstore

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sealingRepository(secret string) *sessionRepository {
	return &sessionRepository{sealKey: hashutils.DeriveKey(secret, sealKeyLabel)}
}

func sealedSession(t *testing.T, repo *sessionRepository) *session.Session {
	t.Helper()

	sess := &session.Session{
		ID:               pulid.MustNew("ses_"),
		UserID:           pulid.MustNew("usr_"),
		OrganizationID:   pulid.MustNew("org_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		SecretHash:       "secret-hash",
		AuthenticatorAAL: 1,
		ExpiresAt:        1_900_000_000,
		CreatedAt:        1_800_000_000,
		LastAccessedAt:   1_800_000_000,
	}
	require.NoError(t, repo.seal(sess))
	return sess
}

func sealMatches(t *testing.T, repo *sessionRepository, sess *session.Session) bool {
	t.Helper()

	want, err := repo.sealOf(sess)
	require.NoError(t, err)
	return want == sess.Seal
}

func TestSealBreaksWhenAProtectedFieldChanges(t *testing.T) {
	t.Parallel()

	repo := sealingRepository("unit-test-session-secret-with-32-bytes")
	for name, tamper := range map[string]func(*session.Session){
		"organization": func(s *session.Session) { s.OrganizationID = pulid.MustNew("org_") },
		"user":         func(s *session.Session) { s.UserID = pulid.MustNew("usr_") },
		"roles":        func(s *session.Session) { s.ActiveRoleIDs = []pulid.ID{pulid.MustNew("rol_")} },
		"assurance":    func(s *session.Session) { s.AuthenticatorAAL = 2 },
		"secret":       func(s *session.Session) { s.SecretHash = "attacker-hash" },
		"grace":        func(s *session.Session) { s.PreviousSecretUntil = 1_999_999_999 },
		"epoch":        func(s *session.Session) { s.SessionEpoch = 9 },
		"lifetime":     func(s *session.Session) { s.ExpiresAt = 1_999_999_999 },
		"issuer":       func(s *session.Session) { s.ExternalIssuer = "https://idp.example.com" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sess := sealedSession(t, repo)
			tamper(sess)

			assert.False(t, sealMatches(t, repo, sess))
		})
	}
}

func TestSealSurvivesActivityAndDisplayFields(t *testing.T) {
	t.Parallel()

	repo := sealingRepository("unit-test-session-secret-with-32-bytes")
	sess := sealedSession(t, repo)
	sess.LastAccessedAt = 1_800_000_600
	sess.AccessConfirmedAt = 1_800_000_600
	sess.Locale = "es"
	sess.UserAgent = "another browser"

	assert.True(t, sealMatches(t, repo, sess))
}

func TestSealTreatsNoRolesAndEmptyRolesAlike(t *testing.T) {
	t.Parallel()

	repo := sealingRepository("unit-test-session-secret-with-32-bytes")
	sess := sealedSession(t, repo)
	sess.ActiveRoleIDs = []pulid.ID{}

	assert.True(t, sealMatches(t, repo, sess))
}

func TestSealNeedsTheServerKey(t *testing.T) {
	t.Parallel()

	sess := sealedSession(t, sealingRepository("unit-test-session-secret-with-32-bytes"))

	assert.False(t, sealMatches(t, sealingRepository("another-session-secret-of-32-bytes!!"), sess))
}
