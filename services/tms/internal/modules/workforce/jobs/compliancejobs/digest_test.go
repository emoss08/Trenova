package compliancejobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func utcAt(year int, month time.Month, day, hour int) int64 {
	return time.Date(year, month, day, hour, 0, 0, 0, time.UTC).Unix()
}

func calendarDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}

func weeklyOn(day time.Weekday) *dashcontrol.DashControl {
	return &dashcontrol.DashControl{
		SendCredentialReminders: true,
		DriverDigestCadence:     dashcontrol.DigestWeekly,
		DriverDigestWeekday:     int16(day),
	}
}

func dailyControl() *dashcontrol.DashControl {
	return &dashcontrol.DashControl{
		SendCredentialReminders: true,
		DriverDigestCadence:     dashcontrol.DigestDaily,
	}
}

func TestDigestDay(t *testing.T) {
	t.Parallel()

	chicago := mustZone(t, "America/Chicago")
	// 2026-09-04 11:00 UTC is Friday 06:00 in Chicago.
	friday := utcAt(2026, time.September, 4, 11)

	t.Run("immediate never bundles", func(t *testing.T) {
		t.Parallel()
		control := &dashcontrol.DashControl{
			SendCredentialReminders: true,
			DriverDigestCadence:     dashcontrol.DigestImmediate,
		}
		day, due := digestDay(control, friday, chicago)
		assert.False(t, due)
		assert.True(t, day.IsZero())
	})

	t.Run("daily fires every night for the local day", func(t *testing.T) {
		t.Parallel()
		day, due := digestDay(dailyControl(), friday, chicago)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 4), day)

		day, due = digestDay(dailyControl(), utcAt(2026, time.September, 5, 11), chicago)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 5), day)
	})

	t.Run("weekly fires only on its own day", func(t *testing.T) {
		t.Parallel()
		day, due := digestDay(weeklyOn(time.Friday), friday, chicago)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 4), day)

		_, due = digestDay(weeklyOn(time.Friday), utcAt(2026, time.September, 5, 11), chicago)
		assert.False(t, due)
	})

	// 2026-09-08 03:00 UTC is Tuesday in UTC and still Monday 22:00 in Chicago.
	t.Run("weekly follows the carrier's weekday, not the UTC one", func(t *testing.T) {
		t.Parallel()
		late := utcAt(2026, time.September, 8, 3)

		day, due := digestDay(weeklyOn(time.Monday), late, chicago)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 7), day)

		_, due = digestDay(weeklyOn(time.Tuesday), late, chicago)
		assert.False(t, due)
	})

	t.Run("reminders switched off stops the digest", func(t *testing.T) {
		t.Parallel()
		control := dailyControl()
		control.SendCredentialReminders = false
		_, due := digestDay(control, friday, chicago)
		assert.False(t, due)
	})

	t.Run("a nil control sends nothing", func(t *testing.T) {
		t.Parallel()
		_, due := digestDay(nil, friday, chicago)
		assert.False(t, due)
	})
}

// US clocks change on 2026-03-08 and 2026-11-01. The sweep's hour moves by one
// local hour but stays on the same local day, so the weekday holds.
func TestDigestDay_USClockChanges(t *testing.T) {
	t.Parallel()

	chicago := mustZone(t, "America/Chicago")

	tests := []struct {
		name string
		now  int64
		want time.Time
	}{
		{
			name: "the evening clocks spring forward",
			now:  utcAt(2026, time.March, 9, 3),
			want: calendarDay(2026, time.March, 8),
		},
		{
			name: "the morning after clocks spring forward",
			now:  utcAt(2026, time.March, 9, 11),
			want: calendarDay(2026, time.March, 9),
		},
		{
			name: "the evening clocks fall back",
			now:  utcAt(2026, time.November, 2, 3),
			want: calendarDay(2026, time.November, 1),
		},
		{
			name: "the morning after clocks fall back",
			now:  utcAt(2026, time.November, 2, 11),
			want: calendarDay(2026, time.November, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, []time.Time{tt.want}, sweepDays(tt.now, chicago))

			day, due := digestDay(weeklyOn(tt.want.Weekday()), tt.now, chicago)
			assert.True(t, due)
			assert.Equal(t, tt.want, day)
		})
	}
}

// Auckland runs the 11:00 UTC sweep within an hour of local midnight, so a clock
// change moves the run across it. Spring forward on 2026-09-27 takes the run
// from Saturday 23:00 to Monday 00:00; fall back on 2026-04-05 takes it from
// Sunday 00:00 to Sunday 23:00. Every local day is still reached exactly once.
func TestDigestDay_ClockChangeAcrossLocalMidnight(t *testing.T) {
	t.Parallel()

	auckland := mustZone(t, "Pacific/Auckland")

	t.Run("spring forward reaches the skipped Sunday", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			[]time.Time{calendarDay(2026, time.September, 26)},
			sweepDays(utcAt(2026, time.September, 26, 11), auckland),
		)
		jump := utcAt(2026, time.September, 27, 11)
		assert.Equal(t,
			[]time.Time{
				calendarDay(2026, time.September, 27),
				calendarDay(2026, time.September, 28),
			},
			sweepDays(jump, auckland),
		)

		day, due := digestDay(weeklyOn(time.Sunday), jump, auckland)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 27), day)

		day, due = digestDay(weeklyOn(time.Monday), jump, auckland)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 28), day)

		day, due = digestDay(dailyControl(), jump, auckland)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.September, 28), day)
	})

	t.Run("fall back does not send Sunday twice", func(t *testing.T) {
		t.Parallel()

		first := utcAt(2026, time.April, 4, 11)
		repeat := utcAt(2026, time.April, 5, 11)

		assert.Equal(t,
			[]time.Time{calendarDay(2026, time.April, 5)},
			sweepDays(first, auckland),
		)
		assert.Empty(t, sweepDays(repeat, auckland))

		day, due := digestDay(weeklyOn(time.Sunday), first, auckland)
		assert.True(t, due)
		assert.Equal(t, calendarDay(2026, time.April, 5), day)

		_, due = digestDay(weeklyOn(time.Sunday), repeat, auckland)
		assert.False(t, due)
		_, due = digestDay(dailyControl(), repeat, auckland)
		assert.False(t, due)

		assert.Equal(t,
			[]time.Time{calendarDay(2026, time.April, 6)},
			sweepDays(utcAt(2026, time.April, 6, 11), auckland),
		)
	})
}

type fakeDashControls struct {
	control *dashcontrol.DashControl
	reads   int
}

func (f *fakeDashControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*dashcontrol.DashControl, error) {
	f.reads++
	return f.control, nil
}

func (f *fakeDashControls) Update(
	context.Context,
	*dashcontrol.DashControl,
) (*dashcontrol.DashControl, error) {
	return nil, errors.New("not used")
}

func TestDigestTenantFor_UsesTheOrganizationsTimezoneAndCaches(t *testing.T) {
	t.Parallel()

	info := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	controls := &fakeDashControls{control: weeklyOn(time.Monday)}
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: info}).
		Return(&tenant.Organization{Timezone: "America/Los_Angeles"}, nil).
		Once()
	activities := &Activities{dashControlRepo: controls, organizations: orgs}
	state := &digestState{
		// Tuesday 02:00 UTC is Monday 19:00 in Los Angeles.
		now:     utcAt(2026, time.September, 8, 2),
		tenants: make(map[pagination.TenantInfo]*digestTenant),
		result:  new(DriverDigestSweepResult),
	}

	entry, err := activities.digestTenantFor(t.Context(), state, info)
	require.NoError(t, err)
	again, err := activities.digestTenantFor(t.Context(), state, info)
	require.NoError(t, err)

	assert.True(t, entry.due)
	assert.Equal(t, calendarDay(2026, time.September, 7), entry.day)
	assert.Same(t, entry, again)
	assert.Equal(t, 1, controls.reads)
	assert.Equal(t, 1, state.result.OrganizationsBundling)
}

func TestDigestTenantFor_SkipsTheTimezoneWhenNothingBundles(t *testing.T) {
	t.Parallel()

	controls := &fakeDashControls{control: &dashcontrol.DashControl{
		SendCredentialReminders: true,
		DriverDigestCadence:     dashcontrol.DigestImmediate,
	}}
	activities := &Activities{
		dashControlRepo: controls,
		organizations:   tenantportsmock.NewMockOrganizationRepository(t),
	}
	state := &digestState{
		now:     utcAt(2026, time.September, 8, 11),
		tenants: make(map[pagination.TenantInfo]*digestTenant),
		result:  new(DriverDigestSweepResult),
	}

	entry, err := activities.digestTenantFor(t.Context(), state, pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})

	require.NoError(t, err)
	assert.False(t, entry.due)
	assert.Equal(t, 0, state.result.OrganizationsBundling)
}

func TestDigestTenantFor_FailsWhenTheOrganizationCannotBeRead(t *testing.T) {
	t.Parallel()

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("down"))
	activities := &Activities{
		dashControlRepo: &fakeDashControls{control: dailyControl()},
		organizations:   orgs,
	}
	state := &digestState{
		now:     utcAt(2026, time.September, 8, 11),
		tenants: make(map[pagination.TenantInfo]*digestTenant),
		result:  new(DriverDigestSweepResult),
	}

	_, err := activities.digestTenantFor(t.Context(), state, pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})

	require.Error(t, err)
	assert.Empty(t, state.tenants)
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
// run a new notice. It names the carrier's local day, not the UTC one.
func TestDigestPeriodKey(t *testing.T) {
	t.Parallel()

	monday := calendarDay(2026, time.September, 7)
	tuesday := calendarDay(2026, time.September, 8)

	assert.Equal(t, "2026-09-07", digestPeriodKey(dashcontrol.DigestDaily, monday))
	assert.Equal(t, "2026-09-08", digestPeriodKey(dashcontrol.DigestDaily, tuesday))
	assert.Equal(t, "2026W37", digestPeriodKey(dashcontrol.DigestWeekly, monday))
	assert.Equal(t, "2026W37", digestPeriodKey(dashcontrol.DigestWeekly, tuesday))

	chicago := mustZone(t, "America/Chicago")
	day, due := digestDay(dailyControl(), utcAt(2026, time.September, 8, 3), chicago)
	require.True(t, due)
	assert.Equal(t, "2026-09-07", digestPeriodKey(dashcontrol.DigestDaily, day))
}

func TestDigestPriority(t *testing.T) {
	t.Parallel()

	upcoming := []notificationtemplate.DriverObligation{{DueInDays: 5}, {DueInDays: 9}}
	assert.Equal(t, notification.PriorityMedium, digestPriority(upcoming))

	// A round-up of things due next week is not the same message as one saying
	// a licence expired on Tuesday.
	lapsed := []notificationtemplate.DriverObligation{
		{DueInDays: 5},
		{DueInDays: -2, Overdue: true},
	}
	assert.Equal(t, notification.PriorityHigh, digestPriority(lapsed))
}

func TestDigestPeriodLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "today", digestPeriodLabel(dashcontrol.DigestDaily))
	assert.Equal(t, "this week", digestPeriodLabel(dashcontrol.DigestWeekly))
}
