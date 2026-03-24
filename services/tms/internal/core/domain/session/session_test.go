package session

import (
	"testing"

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
