package watchtower

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNow = int64(1_700_000_000)

func openItem() *Item {
	return &Item{
		ID:             pulid.MustNew("wt_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		SourceKind:     SourceServiceFailure,
		SourceID:       "sf_1",
		Severity:       SeverityWarning,
		Status:         StatusOpen,
	}
}

func TestAcknowledge_OnlyFromOpen(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.NoError(t, item.Acknowledge(testNow))
	assert.Equal(t, StatusAcknowledged, item.Status)
	require.NotNil(t, item.AcknowledgedAt)
	require.Error(t, item.Acknowledge(testNow))
}

func TestStart_StampsAcknowledgementAndClearsSnooze(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.NoError(t, item.Snooze(testNow+60, testNow))
	require.NoError(t, item.Start(testNow))
	assert.Equal(t, StatusInProgress, item.Status)
	assert.Nil(t, item.SnoozedUntil)
	require.NotNil(t, item.AcknowledgedAt)
	require.Error(t, item.Start(testNow))
}

func TestSnooze_BoundsTheWindow(t *testing.T) {
	t.Parallel()

	item := openItem()
	require.Error(t, item.Snooze(testNow, testNow))
	require.Error(t, item.Snooze(testNow+MaxSnoozeSeconds+1, testNow))
	require.NoError(t, item.Snooze(testNow+MaxSnoozeSeconds, testNow))
	assert.Equal(t, StatusSnoozed, item.Status)
	assert.False(t, item.IsOverdue(testNow+MaxSnoozeSeconds*2))
}

func TestWake_ReturnsToWhereTheItemWas(t *testing.T) {
	t.Parallel()

	fresh := openItem()
	require.NoError(t, fresh.Snooze(testNow+60, testNow))
	assert.True(t, fresh.Wake())
	assert.Equal(t, StatusOpen, fresh.Status)
	assert.False(t, fresh.Wake())

	seen := openItem()
	require.NoError(t, seen.Acknowledge(testNow))
	require.NoError(t, seen.Snooze(testNow+60, testNow))
	assert.True(t, seen.Wake())
	assert.Equal(t, StatusAcknowledged, seen.Status)
}

func TestSetDue_BoundsHowFarOut(t *testing.T) {
	t.Parallel()

	item := openItem()
	tooFar := testNow + MaxDueAheadSeconds + 1
	require.Error(t, item.SetDue(&tooFar, testNow))
	due := testNow + 3600
	require.NoError(t, item.SetDue(&due, testNow))
	assert.True(t, item.IsOverdue(due+1))
	assert.False(t, item.IsOverdue(due-1))
	require.NoError(t, item.SetDue(nil, testNow))
	assert.False(t, item.IsOverdue(due+1))
}

func TestAssign_RefusesANoOp(t *testing.T) {
	t.Parallel()

	item := openItem()
	assignee := pulid.MustNew("usr_")
	require.NoError(t, item.Assign(assignee))
	require.Error(t, item.Assign(assignee))
	require.NoError(t, item.Assign(pulid.Nil))
	assert.True(t, item.AssigneeID.IsNil())
}

func TestResolve_ChecksTheCodeAndNote(t *testing.T) {
	t.Parallel()

	actor := pulid.MustNew("usr_")
	cases := []struct {
		name string
		code ResolutionCode
		note string
		ok   bool
	}{
		{name: "selectable code", code: ResolutionCarrierDelay, ok: true},
		{name: "system code", code: ResolutionCleared},
		{name: "unknown code", code: ResolutionCode("Vanished")},
		{name: "other without note", code: ResolutionOther, note: "  "},
		{name: "other with note", code: ResolutionOther, note: "Phoned in", ok: true},
		{name: "note too long", code: ResolutionFixed, note: strings.Repeat("x", MaxNoteLength+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := openItem()
			err := item.Resolve(tc.code, tc.note, actor, testNow)
			if !tc.ok {
				require.Error(t, err)
				assert.False(t, item.IsResolved())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, StatusResolved, item.Status)
			assert.Equal(t, actor, item.ResolvedByID)
			assert.Equal(t, tc.code, item.ResolutionCode)
		})
	}
}

func TestResolvedItemRefusesWorkUntilReopened(t *testing.T) {
	t.Parallel()

	item := openItem()
	item.EscalationLevel = 2
	require.NoError(t, item.Resolve(ResolutionFixed, "", pulid.MustNew("usr_"), testNow))
	require.Error(t, item.Acknowledge(testNow))
	require.Error(t, item.Assign(pulid.MustNew("usr_")))
	require.Error(t, item.Resolve(ResolutionFixed, "", pulid.Nil, testNow))

	require.NoError(t, item.Reopen())
	assert.Equal(t, StatusOpen, item.Status)
	assert.Nil(t, item.ResolvedAt)
	assert.Empty(t, item.ResolutionCode)
	assert.Zero(t, item.EscalationLevel)
	require.Error(t, item.Reopen())
}

func TestValidateComment_TrimsAndBounds(t *testing.T) {
	t.Parallel()

	body, err := ValidateComment("  on it  ")
	require.NoError(t, err)
	assert.Equal(t, "on it", body)
	_, err = ValidateComment(" ")
	require.Error(t, err)
	_, err = ValidateComment(strings.Repeat("x", MaxCommentLength+1))
	require.Error(t, err)
}

func TestNewEvent_TruncatesLongValues(t *testing.T) {
	t.Parallel()

	item := openItem()
	event := NewEvent(item, EventDueChanged, pulid.Nil).
		WithChange("", strings.Repeat("9", MaxEventValueLength+10))
	assert.Equal(t, item.ID, event.ItemID)
	assert.Equal(t, item.OrganizationID, event.OrganizationID)
	assert.Len(t, event.ToValue, MaxEventValueLength)
}

func TestSeverityRaised(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SeverityWarning, SeverityInfo.Raised())
	assert.Equal(t, SeverityCritical, SeverityWarning.Raised())
	assert.Equal(t, SeverityCritical, SeverityCritical.Raised())
}
