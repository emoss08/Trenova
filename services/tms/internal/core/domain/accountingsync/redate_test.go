package accountingsync_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstOpenDayIsTheDayAfterTheBooksClosedInTheOrganizationsTimezone(t *testing.T) {
	t.Parallel()

	closed := time.Date(2026, time.April, 10, 23, 59, 59, 0, time.UTC).Unix()
	conn := &accountingsync.AccountingConnection{ExternalBooksClosedThrough: &closed}

	for _, zone := range []string{"UTC", "America/Los_Angeles", "Asia/Tokyo", "Pacific/Kiritimati"} {
		loc, err := time.LoadLocation(zone)
		require.NoError(t, err)

		day, ok := conn.FirstOpenDay(loc)

		require.True(t, ok, zone)
		assert.Equal(t, "2026-04-11", timeutils.FormatCalendarDate(day, loc), zone)
		assert.False(t, conn.BooksClosedOn(day), zone)
	}

	_, ok := (&accountingsync.AccountingConnection{}).FirstOpenDay(time.UTC)
	assert.False(t, ok)
}

func TestRedateQueuesOnlyARecordHeldByAClosedPeriod(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	held := &accountingsync.AccountingSyncRecord{
		Status:        accountingsync.SyncStatusBlocked,
		ErrorCategory: accountingsync.SyncErrorClosedPeriod,
		AttemptCount:  3,
	}

	require.NoError(t, held.Redate(actor, 1_775_000_000, 1_776_000_000))

	assert.Equal(t, accountingsync.SyncStatusQueued, held.Status)
	assert.Equal(t, 0, held.AttemptCount)
	require.NotNil(t, held.RedatedTo)
	assert.Equal(t, int64(1_775_000_000), *held.RedatedTo)
	assert.Equal(t, actor, held.RedatedByID)
	require.NotNil(t, held.NextAttemptAt)
	assert.Equal(t, int64(1_776_000_000), *held.NextAttemptAt)
	assert.Equal(t, int64(1_775_000_000), held.SentDate(1_700_000_000))
	assert.Equal(t, int64(1_700_000_000), (&accountingsync.AccountingSyncRecord{}).SentDate(1_700_000_000))

	for _, record := range []*accountingsync.AccountingSyncRecord{
		{Status: accountingsync.SyncStatusBlocked, ErrorCategory: accountingsync.SyncErrorMapping},
		{Status: accountingsync.SyncStatusSynced, ErrorCategory: accountingsync.SyncErrorClosedPeriod},
		{Status: accountingsync.SyncStatusQueued},
	} {
		assert.ErrorIs(t, record.Redate(actor, 1, 2), accountingsync.ErrSyncRecordNotRedatable)
		assert.Nil(t, record.RedatedTo)
	}
}
