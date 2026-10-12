package notification_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	return loc
}

func prefs() *notification.Preference {
	return notification.DefaultPreference(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_"))
}

func input(t *testing.T, eventType string, at time.Time) notification.DeliveryInput {
	t.Helper()
	return notification.DeliveryInput{
		EventType:    eventType,
		Priority:     notification.PriorityMedium,
		Availability: notification.AvailabilityAvailable,
		Now:          at,
		Location:     chicago(t),
	}
}

func wednesdayNoon(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, time.October, 14, 12, 0, 0, 0, chicago(t))
}

func TestDecideModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		desktop     notification.DeliveryMode
		email       notification.DeliveryMode
		digest      bool
		eventType   string
		priority    notification.Priority
		wantDesktop bool
		wantEmail   notification.EmailDelivery
	}{
		{"urgent-only passes an urgent type", notification.DeliveryModeUrgent, notification.DeliveryModeUrgent, true, "tender_waterfall_exhausted", notification.PriorityHigh, true, notification.EmailDeliveryNow},
		{"urgent-only holds a routine type", notification.DeliveryModeUrgent, notification.DeliveryModeUrgent, true, "tender_accepted", notification.PriorityHigh, false, notification.EmailDeliveryNone},
		{"critical priority is urgent whatever the type", notification.DeliveryModeUrgent, notification.DeliveryModeUrgent, false, "agent.quality_regression", notification.PriorityCritical, true, notification.EmailDeliveryNow},
		{"everything without a digest emails now", notification.DeliveryModeAll, notification.DeliveryModeAll, false, "tender_accepted", notification.PriorityMedium, true, notification.EmailDeliveryNow},
		{"everything with a digest bundles routine ones", notification.DeliveryModeAll, notification.DeliveryModeAll, true, "tender_accepted", notification.PriorityMedium, true, notification.EmailDeliveryDigest},
		{"a digest still sends urgent ones right away", notification.DeliveryModeAll, notification.DeliveryModeAll, true, "carrier_intel_block", notification.PriorityHigh, true, notification.EmailDeliveryNow},
		{"off holds even urgent ones", notification.DeliveryModeOff, notification.DeliveryModeOff, false, "carrier_intel_block", notification.PriorityCritical, false, notification.EmailDeliveryNone},
		{"the channels are independent", notification.DeliveryModeOff, notification.DeliveryModeAll, false, "tender_accepted", notification.PriorityMedium, false, notification.EmailDeliveryNow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := prefs()
			p.DesktopMode = tt.desktop
			p.EmailMode = tt.email
			p.EmailDigest = tt.digest
			in := input(t, tt.eventType, wednesdayNoon(t))
			in.Priority = tt.priority

			got := p.Decide(in)

			assert.Equal(t, tt.wantDesktop, got.Desktop)
			assert.Equal(t, tt.wantEmail, got.Email)
		})
	}
}

func TestDecideSoundFollowsDesktop(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DesktopMode = notification.DeliveryModeAll
	p.Sound = true
	assert.True(t, p.Decide(input(t, "tender_accepted", wednesdayNoon(t))).Sound)

	p.Sound = false
	assert.False(t, p.Decide(input(t, "tender_accepted", wednesdayNoon(t))).Sound)

	p.Sound = true
	p.DesktopMode = notification.DeliveryModeOff
	assert.False(t, p.Decide(input(t, "tender_accepted", wednesdayNoon(t))).Sound)
}

func TestDecideDisabledTypes(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DesktopMode = notification.DeliveryModeAll
	p.EmailMode = notification.DeliveryModeAll
	p.DisabledEventTypes = []string{"tender_accepted", notification.TCAPreferenceKey}

	off := p.Decide(input(t, "tender_accepted", wednesdayNoon(t)))
	assert.False(t, off.Desktop)
	assert.Equal(t, notification.EmailDeliveryNone, off.Email)
	assert.Equal(t, notification.HoldTypeDisabled, off.DesktopHold)

	assert.False(t, p.Decide(input(t, "tca.update", wednesdayNoon(t))).Desktop)
	assert.True(t, p.Decide(input(t, "tender_needs_review", wednesdayNoon(t))).Desktop)
}

func TestDecideRequiredTypeIgnoresDisabledList(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DisabledEventTypes = []string{notification.AIAuditChainMismatchEvent}

	got := p.Decide(input(t, notification.AIAuditChainMismatchEvent, wednesdayNoon(t)))

	assert.True(t, got.Desktop)
	assert.True(t, got.Urgent)
}

func TestDecidePause(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DesktopMode = notification.DeliveryModeAll
	until := wednesdayNoon(t).Add(time.Hour).Unix()
	p.PausedUntil = &until

	paused := p.Decide(input(t, "carrier_intel_block", wednesdayNoon(t)))
	assert.False(t, paused.Desktop)
	assert.Equal(t, notification.HoldPaused, paused.DesktopHold)

	later := p.Decide(input(t, "carrier_intel_block", wednesdayNoon(t).Add(2*time.Hour)))
	assert.True(t, later.Desktop)
}

func TestDecideAvailability(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DesktopMode = notification.DeliveryModeAll

	focus := input(t, "tender_accepted", wednesdayNoon(t))
	focus.Availability = notification.AvailabilityFocus
	assert.Equal(t, notification.HoldFocus, p.Decide(focus).DesktopHold)

	focus.EventType = notification.MentionEventType
	assert.True(t, p.Decide(focus).Desktop, "a direct mention reaches someone focusing")

	focus.EventType = "tender_waterfall_exhausted"
	assert.True(t, p.Decide(focus).Desktop, "urgent reaches someone focusing")

	away := input(t, notification.MentionEventType, wednesdayNoon(t))
	away.Availability = notification.AvailabilityAway
	assert.Equal(t, notification.HoldAway, p.Decide(away).DesktopHold, "mentions go to the hand-off")

	away.EventType = "carrier_intel_block"
	assert.True(t, p.Decide(away).Desktop)
}

func TestInQuietHoursOvernight(t *testing.T) {
	t.Parallel()
	loc := chicago(t)
	p := prefs()
	p.QuietHoursEnabled = true
	p.QuietHoursStart = "19:00"
	p.QuietHoursEnd = "07:00"
	p.QuietHoursDays = []int{1, 2, 3, 4, 5}

	at := func(day time.Weekday, hour, minute int) time.Time {
		base := time.Date(2026, time.October, 11, hour, minute, 0, 0, loc)
		return base.AddDate(0, 0, int(day))
	}

	assert.True(t, p.InQuietHours(at(time.Monday, 19, 0), loc), "starts on the hour")
	assert.True(t, p.InQuietHours(at(time.Tuesday, 6, 59), loc), "Monday's night runs into Tuesday")
	assert.False(t, p.InQuietHours(at(time.Tuesday, 7, 0), loc), "ends on the hour")
	assert.True(t, p.InQuietHours(at(time.Saturday, 3, 0), loc), "Friday's night runs into Saturday")
	assert.False(t, p.InQuietHours(at(time.Saturday, 20, 0), loc), "Saturday is not a quiet day")
	assert.False(t, p.InQuietHours(at(time.Monday, 3, 0), loc), "Sunday's night is not quiet")
	assert.False(t, p.InQuietHours(at(time.Wednesday, 12, 0), loc))

	fridayLateUTC := time.Date(2026, time.October, 16, 23, 30, 0, 0, time.UTC)
	assert.False(t, p.InQuietHours(fridayLateUTC, loc), "18:30 in Chicago is before quiet hours")
	assert.True(t, p.InQuietHours(fridayLateUTC, time.UTC), "the same instant is 23:30 in UTC")
}

func TestInQuietHoursSameDay(t *testing.T) {
	t.Parallel()
	loc := chicago(t)
	p := prefs()
	p.QuietHoursEnabled = true
	p.QuietHoursStart = "12:00"
	p.QuietHoursEnd = "13:30"
	p.QuietHoursDays = []int{3}

	assert.True(t, p.InQuietHours(time.Date(2026, time.October, 14, 12, 45, 0, 0, loc), loc))
	assert.False(t, p.InQuietHours(time.Date(2026, time.October, 14, 13, 30, 0, 0, loc), loc))
	assert.False(t, p.InQuietHours(time.Date(2026, time.October, 15, 12, 45, 0, 0, loc), loc))

	p.QuietHoursEnabled = false
	assert.False(t, p.InQuietHours(time.Date(2026, time.October, 14, 12, 45, 0, 0, loc), loc))
}

func TestDecideQuietHoursLetsUrgentThroughOnlyWhenAllowed(t *testing.T) {
	t.Parallel()
	loc := chicago(t)
	p := prefs()
	p.DesktopMode = notification.DeliveryModeAll
	p.QuietHoursEnabled = true
	p.QuietHoursDays = []int{0, 1, 2, 3, 4, 5, 6}
	night := time.Date(2026, time.October, 14, 22, 0, 0, 0, loc)

	routine := p.Decide(input(t, "tender_accepted", night))
	assert.Equal(t, notification.HoldQuietHours, routine.DesktopHold)

	p.QuietHoursAllowUrgent = true
	assert.True(t, p.Decide(input(t, "tender_waterfall_exhausted", night)).Desktop)

	p.QuietHoursAllowUrgent = false
	assert.False(t, p.Decide(input(t, "tender_waterfall_exhausted", night)).Desktop)
}

func TestPreferenceValidate(t *testing.T) {
	t.Parallel()

	p := prefs()
	p.DesktopMode = "sometimes"
	p.QuietHoursStart = "25:00"
	p.QuietHoursEnabled = true
	p.QuietHoursDays = []int{7}
	p.DisabledEventTypes = []string{"Tender Accepted", notification.AIAuditChainMismatchEvent}

	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	assert.ElementsMatch(t, []string{
		"desktopMode",
		"quietHoursStart",
		"quietHoursDays[0]",
		"disabledEventTypes[0]",
		"disabledEventTypes[1]",
	}, fields)
}

func TestPreferenceNormalize(t *testing.T) {
	t.Parallel()
	p := prefs()
	p.DisabledEventTypes = []string{" tender_accepted ", "tca.*", "tender_accepted", ""}
	p.QuietHoursDays = []int{5, 1, 5, 3}

	p.Normalize()

	assert.Equal(t, []string{"tca.*", "tender_accepted"}, p.DisabledEventTypes)
	assert.Equal(t, []int{1, 3, 5}, p.QuietHoursDays)

	valid := prefs()
	valid.Normalize()
	multiErr := errortypes.NewMultiError()
	valid.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}
