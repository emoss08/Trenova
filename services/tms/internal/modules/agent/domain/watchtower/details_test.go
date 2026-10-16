package watchtower

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFacts_DropsEmptyFactsAndCapsTheList(t *testing.T) {
	t.Parallel()

	facts := []Fact{
		{Label: " Customer ", Value: " Northline Foods "},
		{Label: "Appointment", At: testNow},
		{Label: "Empty"},
		{Label: "", Value: "no label"},
	}
	for i := range 10 {
		facts = append(facts, Fact{Label: "Extra", Value: strings.Repeat("x", i+1)})
	}

	got := NormalizeFacts(facts)
	require.Len(t, got, MaxFacts)
	assert.Equal(t, Fact{Label: "Customer", Value: "Northline Foods"}, got[0])
	assert.Equal(t, Fact{Label: "Appointment", At: testNow}, got[1])
}

func TestNormalizeTrail_KeepsTheLatestEvents(t *testing.T) {
	t.Parallel()

	trail := make([]TrailEvent, 0, MaxTrail+2)
	for i := range MaxTrail + 2 {
		trail = append(trail, TrailEvent{At: int64(i + 1), Text: "event"})
	}

	got := NormalizeTrail(trail)
	require.Len(t, got, MaxTrail)
	assert.Equal(t, int64(3), got[0].At, "the oldest events are the ones cut")
	assert.Equal(t, int64(MaxTrail+2), got[MaxTrail-1].At)
}

func TestNormalizeImpact_TrimsAndCaps(t *testing.T) {
	t.Parallel()

	got := NormalizeImpact([]string{" 4 tendered loads ", "", "a", "b", "c", "d", "e", "f"})
	assert.Equal(t, []string{"4 tendered loads", "a", "b", "c", "d", "e"}, got)
}

func TestApplyDeadline_MakesTheSourcesDeadlineTheDueTime(t *testing.T) {
	t.Parallel()

	item := openItem()
	item.ApplyDeadline(&Deadline{At: testNow + 3600, Label: "Minneapolis dock closes"})

	require.NotNil(t, item.DueAt)
	assert.Equal(t, testNow+3600, *item.DueAt)
	assert.Equal(t, "Minneapolis dock closes", item.DueLabel)
	assert.Equal(t, DueSourceSource, item.DueSource)

	untouched := openItem()
	untouched.ApplyDeadline(nil)
	untouched.ApplyDeadline(&Deadline{})
	assert.Nil(t, untouched.DueAt)
	assert.Empty(t, untouched.DueSource)
}

func TestSetDue_APersonsDueTimeDropsTheSourcesLabel(t *testing.T) {
	t.Parallel()

	item := openItem()
	item.ApplyDeadline(&Deadline{At: testNow + 3600, Label: "Dock closes"})
	require.NoError(t, item.SetDue(nil, testNow))

	assert.Nil(t, item.DueAt)
	assert.Empty(t, item.DueLabel)
	assert.Equal(t, DueSourcePerson, item.DueSource)
}

func TestValidate_ALabelBelongsOnlyToASourceDeadline(t *testing.T) {
	t.Parallel()

	item := openItem()
	item.Title = "Late load"
	item.OccurredAt = testNow
	item.DueLabel = "Dock closes"
	item.DueSource = DueSourcePolicy

	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	item.DueSource = DueSourceSource
	multiErr = errortypes.NewMultiError()
	item.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), multiErr.Error())
}

func TestUnsnooze_WakesOnlyASnoozedException(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.Error(t, item.Unsnooze())

	require.NoError(t, item.Snooze(testNow+600, testNow))
	require.NoError(t, item.Unsnooze())
	assert.Equal(t, StatusOpen, item.Status)
	assert.Nil(t, item.SnoozedUntil)
}
