package notification_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const subjectNow int64 = 1_800_000_000

func openRow(target pulid.ID) *notification.Notification {
	return &notification.Notification{
		TargetUserID:    &target,
		SubjectKey:      notification.SubjectKey("watchtower", "critical", "MissedPickup"),
		EventType:       "watchtower.critical",
		Priority:        notification.PriorityHigh,
		Title:           "Missed pickup",
		Message:         "Reno Plant closed with no arrival recorded",
		Reason:          notification.ReasonTeam,
		OccurrenceCount: 1,
		SurfacesAt:      subjectNow - 600,
		ReadAt:          new(subjectNow - 300),
	}
}

func recurrence(label string, at int64) *notification.Notification {
	n := &notification.Notification{
		EventType: "watchtower.critical",
		Priority:  notification.PriorityHigh,
		Title:     "Missed pickup",
		Message:   label + " closed with no arrival recorded",
		Reason:    notification.ReasonTeam,
	}

	return n.WithOccurrence(notification.Occurrence{Label: label, Link: "/desk/needs", At: at})
}

func recentLabels(t *testing.T, n *notification.Notification) []string {
	t.Helper()
	recent, ok := n.Data[notification.DataKeyRecent].([]any)
	require.True(t, ok, "the row carries its recent occurrences")
	labels := make([]string, 0, len(recent))
	for _, entry := range recent {
		occurrence, isMap := entry.(map[string]any)
		require.True(t, isMap)
		labels = append(labels, occurrence["label"].(string))
	}

	return labels
}

func TestSubjectKeysJoinTheirPartsAndADayKeyEndsWithTheUTCDay(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "agent:proposals:agt_1", notification.SubjectKey("agent", "proposals", "agt_1"))
	assert.Equal(
		t,
		"watchtower:critical:MissedPickup:2027-01-15",
		notification.DaySubjectKey(subjectNow, "watchtower", "critical", "MissedPickup"),
	)
}

func TestOnlyANotificationWithASubjectAndAPersonCollapses(t *testing.T) {
	t.Parallel()

	target := pulid.MustNew("usr_")
	assert.True(t, openRow(target).Collapses())

	noSubject := openRow(target)
	noSubject.SubjectKey = ""
	assert.False(t, noSubject.Collapses())

	broadcast := openRow(target)
	broadcast.TargetUserID = nil
	assert.False(t, broadcast.Collapses())
}

func TestAbsorbTakesTheLatestContentCountsItAndComesBackUnread(t *testing.T) {
	t.Parallel()

	row := openRow(pulid.MustNew("usr_"))
	next := recurrence("Ontario DC", subjectNow)
	next.Priority = notification.PriorityCritical

	row.Absorb(next, subjectNow)

	assert.Equal(t, "Ontario DC closed with no arrival recorded", row.Message)
	assert.Equal(t, notification.PriorityCritical, row.Priority)
	assert.Equal(t, 2, row.OccurrenceCount)
	assert.Nil(t, row.ReadAt, "a subject that came up again is unread again")
	assert.Equal(t, subjectNow, row.SurfacesAt, "and moves to the top")
	assert.Equal(t, []string{"Ontario DC"}, recentLabels(t, row))
	assert.NotContains(t, row.Data, notification.DataKeyOccurrence)
}

func TestAbsorbKeepsTheNewestThreeOccurrencesNewestFirst(t *testing.T) {
	t.Parallel()

	row := openRow(pulid.MustNew("usr_"))
	for i, label := range []string{"Reno", "Ontario", "Phoenix", "Tampa"} {
		row.Absorb(recurrence(label, subjectNow+int64(i)), subjectNow+int64(i))
	}

	assert.Equal(t, []string{"Tampa", "Phoenix", "Ontario"}, recentLabels(t, row))
	assert.Equal(t, 5, row.OccurrenceCount)
}

func TestAbsorbCountsATallyAndLeavesTheCountAloneForAReminder(t *testing.T) {
	t.Parallel()

	row := openRow(pulid.MustNew("usr_"))

	batch := recurrence("Six more proposals", subjectNow)
	batch.OccurrenceCount = 6
	row.Absorb(batch, subjectNow)
	assert.Equal(t, 7, row.OccurrenceCount)

	reminder := recurrence("Still waiting", subjectNow+60)
	reminder.OccurrenceCount = notification.NoNewOccurrence
	row.Absorb(reminder, subjectNow+60)
	assert.Equal(t, 7, row.OccurrenceCount, "a reminder is not a new occurrence")
	assert.Nil(t, row.ReadAt, "but it does make the row unread")
}

func TestAbsorbLeavesARowSetAsideUntilItsTimeUnlessTheNewsIsCritical(t *testing.T) {
	t.Parallel()

	until := subjectNow + 3_600

	routine := openRow(pulid.MustNew("usr_"))
	routine.SurfacesAt = until
	routine.Absorb(recurrence("Ontario DC", subjectNow), subjectNow)
	assert.Equal(t, until, routine.SurfacesAt, "the person asked not to see it until then")
	assert.True(t, routine.Snoozed(subjectNow))

	urgent := openRow(pulid.MustNew("usr_"))
	urgent.SurfacesAt = until
	critical := recurrence("Ontario DC", subjectNow)
	critical.Priority = notification.PriorityCritical
	urgent.Absorb(critical, subjectNow)
	assert.Equal(t, subjectNow, urgent.SurfacesAt, "critical news does not wait")
	assert.False(t, urgent.Snoozed(subjectNow))
}

func TestAFinishedRowIsNotSnoozedWhateverItsTime(t *testing.T) {
	t.Parallel()

	row := openRow(pulid.MustNew("usr_"))
	row.SurfacesAt = subjectNow + 3_600
	row.DismissedAt = new(subjectNow)

	assert.False(t, row.Snoozed(subjectNow))
}

func TestLensesParseOnlyTheNamesThePanelUses(t *testing.T) {
	t.Parallel()

	for _, lens := range []notification.Lens{
		notification.LensForYou,
		notification.LensTeam,
		notification.LensLater,
		notification.LensDone,
		notification.LensOpen,
	} {
		assert.True(t, lens.IsValid(), string(lens))
	}
	assert.False(t, notification.Lens("inbox").IsValid())
	assert.False(t, notification.Lens("").IsValid())
}
