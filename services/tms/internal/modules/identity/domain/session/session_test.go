package session

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	expiresAt := timeutils.NowUnix() + 3600

	req := &NewSessionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  orgID,
			BuID:   buID,
			UserID: userID,
		},
		ExpiresAt: expiresAt,
	}

	s := NewSession(req)

	assert.False(t, s.ID.IsNil())
	assert.Equal(t, orgID, s.OrganizationID)
	assert.Equal(t, buID, s.BusinessUnitID)
	assert.Equal(t, userID, s.UserID)
	assert.Equal(t, expiresAt, s.ExpiresAt)
	assert.NotZero(t, s.CreatedAt)
	assert.NotZero(t, s.UpdatedAt)
	assert.NotZero(t, s.LastAccessedAt)
}

func TestSession_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid session passes", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ID:        pulid.MustNew("ses_"),
			ExpiresAt: timeutils.NowUnix() + 3600,
		}

		err := s.Validate()
		require.NoError(t, err)
	})

	t.Run("expired session fails", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ID:        pulid.MustNew("ses_"),
			ExpiresAt: timeutils.NowUnix() - 3600,
		}

		err := s.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Session has expired")
	})
}

func TestSession_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("future expiry is not expired", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ExpiresAt: timeutils.NowUnix() + 3600,
		}

		assert.False(t, s.IsExpired())
	})

	t.Run("past expiry is expired", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ExpiresAt: timeutils.NowUnix() - 3600,
		}

		assert.True(t, s.IsExpired())
	})
}

func TestSession_IsValid(t *testing.T) {
	t.Parallel()

	t.Run("non-expired session is valid", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ExpiresAt: timeutils.NowUnix() + 3600,
		}

		assert.True(t, s.IsValid())
	})

	t.Run("expired session is not valid", func(t *testing.T) {
		t.Parallel()

		s := &Session{
			ExpiresAt: timeutils.NowUnix() - 3600,
		}

		assert.False(t, s.IsValid())
	})
}

func TestSession_UpdateLastAccessedAt(t *testing.T) {
	t.Parallel()

	s := &Session{}
	assert.Zero(t, s.LastAccessedAt)
	assert.Zero(t, s.UpdatedAt)

	s.UpdateLastAccessedAt()

	assert.NotZero(t, s.LastAccessedAt)
	assert.Equal(t, s.LastAccessedAt, s.UpdatedAt)
}

func TestPolicy_ExpiresAt(t *testing.T) {
	t.Parallel()

	policy := Policy{MaxAge: 12 * time.Hour}

	assert.Equal(t, int64(1_000+12*60*60), policy.ExpiresAt(1_000))
}

func TestSession_Bound(t *testing.T) {
	t.Parallel()

	t.Run("caps a session issued under a longer lifetime", func(t *testing.T) {
		t.Parallel()

		s := &Session{CreatedAt: 1_000, ExpiresAt: 1_000 + 30*24*60*60}
		s.Bound(Policy{MaxAge: 12 * time.Hour})

		assert.Equal(t, int64(1_000+12*60*60), s.ExpiresAt)
	})

	t.Run("keeps an expiry already inside the lifetime", func(t *testing.T) {
		t.Parallel()

		s := &Session{CreatedAt: 1_000, ExpiresAt: 1_000 + 60*60}
		s.Bound(Policy{MaxAge: 12 * time.Hour})

		assert.Equal(t, int64(1_000+60*60), s.ExpiresAt)
	})

	t.Run("leaves the expiry alone without a lifetime", func(t *testing.T) {
		t.Parallel()

		s := &Session{CreatedAt: 1_000, ExpiresAt: 99_999}
		s.Bound(Policy{})

		assert.Equal(t, int64(99_999), s.ExpiresAt)
	})
}

func TestSession_IdleExpiresAt(t *testing.T) {
	t.Parallel()

	s := &Session{LastAccessedAt: 10_000}

	assert.Zero(t, s.IdleExpiresAt(Policy{MaxAge: time.Hour}))
	assert.Equal(
		t,
		int64(10_000+30*60)+int64(IdleGrace/time.Second),
		s.IdleExpiresAt(Policy{MaxAge: time.Hour, IdleTimeout: 30 * time.Minute}),
	)
}

func TestSession_IsExpiredAt(t *testing.T) {
	t.Parallel()

	policy := Policy{MaxAge: 12 * time.Hour, IdleTimeout: 30 * time.Minute}
	const createdAt int64 = 100_000
	idleWindow := int64((30*time.Minute + IdleGrace) / time.Second)

	tests := []struct {
		name    string
		session Session
		policy  Policy
		now     int64
		expired bool
	}{
		{
			name: "active session inside every limit",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt + 60,
				ExpiresAt:      createdAt + 12*60*60,
			},
			policy:  policy,
			now:     createdAt + 120,
			expired: false,
		},
		{
			name: "idle exactly to the end of its window",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt,
				ExpiresAt:      createdAt + 12*60*60,
			},
			policy:  policy,
			now:     createdAt + idleWindow,
			expired: false,
		},
		{
			name: "idle past its window",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt,
				ExpiresAt:      createdAt + 12*60*60,
			},
			policy:  policy,
			now:     createdAt + idleWindow + 1,
			expired: true,
		},
		{
			name: "idle past the window with the idle timeout turned off",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt,
				ExpiresAt:      createdAt + 12*60*60,
			},
			policy:  Policy{MaxAge: 12 * time.Hour},
			now:     createdAt + idleWindow + 1,
			expired: false,
		},
		{
			name: "issued under a thirty day lifetime and older than the current one",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt + 12*60*60,
				ExpiresAt:      createdAt + 30*24*60*60,
			},
			policy:  policy,
			now:     createdAt + 12*60*60 + 1,
			expired: true,
		},
		{
			name: "past its own expiry",
			session: Session{
				CreatedAt:      createdAt,
				LastAccessedAt: createdAt + 60,
				ExpiresAt:      createdAt + 90,
			},
			policy:  policy,
			now:     createdAt + 91,
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expired, tt.session.IsExpiredAt(tt.policy, tt.now))
			if tt.expired {
				require.Error(t, tt.session.ValidateAt(tt.policy, tt.now))
			} else {
				require.NoError(t, tt.session.ValidateAt(tt.policy, tt.now))
			}
		})
	}
}

func TestKeepAliveIntervalFitsInsideIdleGrace(t *testing.T) {
	t.Parallel()

	assert.Less(t, KeepAliveInterval, IdleGrace)
}
