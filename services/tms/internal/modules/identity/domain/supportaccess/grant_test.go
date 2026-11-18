package supportaccess_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func liveGrant(now int64) *supportaccess.Grant {
	return &supportaccess.Grant{
		ID:            pulid.MustNew(supportaccess.GrantIDPrefix),
		Scope:         supportaccess.ScopeReadOnly,
		DurationHours: 1,
		ExpiresAt:     now + 3600,
		CreatedAt:     now,
	}
}

func TestGrantExtend_CountsTheNewLengthFromNow(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	grant := liveGrant(now - 1800)
	by := pulid.MustNew("usr_")

	require.NoError(t, grant.Extend(by, 24, now))
	assert.Equal(t, now+24*3600, grant.ExpiresAt)
	assert.Equal(t, 24, grant.DurationHours)
	require.NotNil(t, grant.ExtendedAt)
	assert.Equal(t, now, *grant.ExtendedAt)
	assert.Equal(t, by, grant.ExtendedByID)
}

func TestGrantExtend_RefusesAnEndedGrantALengthNotOfferedOrAShorterEnd(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	by := pulid.MustNew("usr_")

	ended := liveGrant(now - 7200)
	require.Error(t, ended.Extend(by, 4, now), "an ended grant is granted again, not extended")

	revoked := liveGrant(now)
	revokedAt := now
	revoked.RevokedAt = &revokedAt
	require.Error(t, revoked.Extend(by, 4, now))

	odd := liveGrant(now)
	require.Error(t, odd.Extend(by, 5, now))

	long := liveGrant(now)
	long.ExpiresAt = now + 24*3600
	require.Error(t, long.Extend(by, 1, now), "extending never shortens access")
	assert.Equal(t, now+24*3600, long.ExpiresAt)
	assert.Nil(t, long.ExtendedAt)
}

func TestGrantIsFirstEntryOnlyBeforeStaffEnter(t *testing.T) {
	t.Parallel()

	grant := liveGrant(1_800_000_000)
	assert.True(t, grant.IsFirstEntry())
	at := int64(1_800_000_100)
	grant.LastEnteredAt = &at
	assert.False(t, grant.IsFirstEntry())
}
