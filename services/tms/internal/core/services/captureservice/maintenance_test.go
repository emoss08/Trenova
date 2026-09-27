package captureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stack(w *world, status capture.BatchStatus, retainUntil int64) *capture.CaptureBatch {
	batch := &capture.CaptureBatch{
		ID:             pulid.MustNew("cbat_"),
		OrganizationID: w.tenant.OrgID,
		BusinessUnitID: w.tenant.BuID,
		UserID:         w.tenant.UserID,
		DeviceID:       pulid.MustNew("cdev_"),
		Source:         capture.SourceScan,
		SourceName:     "fi-7160",
		Status:         status,
		ItemCount:      3,
		FiledItemCount: 1,
		RetainUntil:    retainUntil,
	}
	w.batches[batch.ID] = batch

	return batch
}

func TestRetentionRemindsTheOwnerOnceAWeekAhead(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	now := timeutils.NowUnix()
	day := int64(secondsPerDay)

	due := stack(w, capture.BatchPartiallyFiled, now+3*day-60)
	stack(w, capture.BatchFiled, now+3*day)
	stack(w, capture.BatchReady, now+10*day)
	stack(w, capture.BatchReady, now-day)

	reminded, err := s.RemindRetention(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, reminded, "only a stack still waiting, and due within the week")
	require.Len(t, w.notified, 1)

	sent := w.notified[0]
	assert.Equal(t, w.tenant.UserID, *sent.TargetUserID)
	assert.Equal(t, notification.ChannelUser, sent.Channel)
	assert.Equal(t, retentionReminderEvent, sent.EventType)
	assert.Contains(t, sent.Message, "A scan from fi-7160 has 2 document(s) not filed yet")
	assert.Contains(t, sent.Message, "deleted in 3 day(s)")
	assert.Contains(t, sent.Data["link"], due.ID.String())
	assert.NotNil(t, w.batches[due.ID].RetentionRemindedAt)

	reminded, err = s.RemindRetention(t.Context())
	require.NoError(t, err)
	assert.Zero(t, reminded, "an owner is told once")
	assert.Len(t, w.notified, 1)
}

func TestRetentionRemindersWaitForNotifications(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	s.notifications = nil
	stack(w, capture.BatchReady, timeutils.NowUnix()+secondsPerDay)

	reminded, err := s.RemindRetention(t.Context())
	require.NoError(t, err)
	assert.Zero(t, reminded)
	for _, batch := range w.batches {
		assert.Nil(t, batch.RetentionRemindedAt, "nobody was told, so nothing is marked told")
	}
}
