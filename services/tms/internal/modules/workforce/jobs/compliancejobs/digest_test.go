package compliancejobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/stretchr/testify/assert"
)

func digestAt(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 3, 0, 0, 0, time.UTC).Unix()
}

func TestDigestDueToday(t *testing.T) {
	t.Parallel()

	// 2026-09-04 is a Friday.
	friday := digestAt(2026, time.September, 4)

	t.Run("immediate never bundles", func(t *testing.T) {
		t.Parallel()
		control := &dashcontrol.DashControl{
			SendCredentialReminders: true,
			DriverDigestCadence:     dashcontrol.DigestImmediate,
		}
		assert.False(t, digestDueToday(control, friday))
	})

	t.Run("daily fires every night", func(t *testing.T) {
		t.Parallel()
		control := &dashcontrol.DashControl{
			SendCredentialReminders: true,
			DriverDigestCadence:     dashcontrol.DigestDaily,
		}
		assert.True(t, digestDueToday(control, friday))
		assert.True(t, digestDueToday(control, digestAt(2026, time.September, 5)))
	})

	// A weekly digest that fired every night would not be a weekly one.
	t.Run("weekly fires only on its own day", func(t *testing.T) {
		t.Parallel()
		control := &dashcontrol.DashControl{
			SendCredentialReminders: true,
			DriverDigestCadence:     dashcontrol.DigestWeekly,
			DriverDigestWeekday:     int16(time.Friday),
		}
		assert.True(t, digestDueToday(control, friday))
		assert.False(t, digestDueToday(control, digestAt(2026, time.September, 5)))
	})

	// Reminders off means there is nothing to bundle. Bundling anyway would
	// route around a switch somebody deliberately turned off.
	t.Run("reminders switched off stops the digest", func(t *testing.T) {
		t.Parallel()
		control := &dashcontrol.DashControl{
			SendCredentialReminders: false,
			DriverDigestCadence:     dashcontrol.DigestDaily,
		}
		assert.False(t, digestDueToday(control, friday))
	})
}

// A weekly notice has to cover the gap until the next one, or a renewal that
// falls due on day nine is never mentioned before it lapses.
func TestDigestHorizon(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(7), digestHorizon(dashcontrol.DigestDaily))
	assert.Equal(t, int64(14), digestHorizon(dashcontrol.DigestWeekly))
	assert.Equal(t, int64(7), digestHorizon(dashcontrol.DigestImmediate))
}

// The period key is what makes a retried activity idempotent and tomorrow's
// run a new notice.
func TestDigestPeriodKey(t *testing.T) {
	t.Parallel()

	monday := digestAt(2026, time.September, 7)
	tuesday := digestAt(2026, time.September, 8)

	assert.Equal(t, "2026-09-07", digestPeriodKey(dashcontrol.DigestDaily, monday))
	assert.NotEqual(t,
		digestPeriodKey(dashcontrol.DigestDaily, monday),
		digestPeriodKey(dashcontrol.DigestDaily, tuesday),
	)

	// Two days inside one ISO week resolve to the same key, so a retry the
	// next morning does not send a second weekly notice.
	assert.Equal(t,
		digestPeriodKey(dashcontrol.DigestWeekly, monday),
		digestPeriodKey(dashcontrol.DigestWeekly, tuesday),
	)
}

func TestDigestPriority(t *testing.T) {
	t.Parallel()

	upcoming := []documenttemplate.DriverObligation{{DueInDays: 5}, {DueInDays: 9}}
	assert.Equal(t, notification.PriorityMedium, digestPriority(upcoming))

	// A round-up of things due next week is not the same message as one saying
	// a licence expired on Tuesday.
	lapsed := []documenttemplate.DriverObligation{{DueInDays: 5}, {DueInDays: -2, Overdue: true}}
	assert.Equal(t, notification.PriorityHigh, digestPriority(lapsed))
}

func TestDigestPeriodLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "today", digestPeriodLabel(dashcontrol.DigestDaily))
	assert.Equal(t, "this week", digestPeriodLabel(dashcontrol.DigestWeekly))
}
