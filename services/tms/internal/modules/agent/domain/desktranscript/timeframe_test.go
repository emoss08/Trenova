package desktranscript_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
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

func TestTimeframe_LastDaysStartAtMidnightInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	now := time.Date(2026, 3, 20, 1, 30, 0, 0, time.UTC)

	cases := map[desktranscript.TimeframePreset]time.Time{
		desktranscript.TimeframeLast7Days:    time.Date(2026, 3, 13, 0, 0, 0, 0, loc),
		desktranscript.TimeframeLast30Days:   time.Date(2026, 2, 18, 0, 0, 0, 0, loc),
		desktranscript.TimeframeLast90Days:   time.Date(2025, 12, 20, 0, 0, 0, 0, loc),
		desktranscript.TimeframeLast12Months: time.Date(2025, 3, 20, 0, 0, 0, 0, loc),
	}
	for preset, from := range cases {
		timeframe := desktranscript.Timeframe{Preset: preset}
		window := timeframe.Resolve(now, loc)
		assert.Equal(t, from.Unix(), window.From, preset)
		assert.Zero(t, window.Before, preset)
	}
}

func TestTimeframe_AllTimeIsUnbounded(t *testing.T) {
	t.Parallel()

	timeframe := desktranscript.AllTime()
	window := timeframe.Resolve(time.Now(), time.UTC)

	assert.False(t, window.Bounded())
}

func TestTimeframe_CustomRunsFromTheFirstDayToTheEndOfTheLast(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	timeframe := desktranscript.Timeframe{
		Preset:   desktranscript.TimeframeCustom,
		FromDate: "2026-03-01",
		ToDate:   "2026-03-31",
	}

	window := timeframe.Resolve(time.Now(), loc)

	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, loc).Unix(), window.From)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, loc).Unix(), window.Before)
}

func TestTimeframe_RefusesDatesOutsideACustomRangeAndABackwardsRange(t *testing.T) {
	t.Parallel()

	withDates := desktranscript.Timeframe{
		Preset:   desktranscript.TimeframeLast7Days,
		FromDate: "2026-03-01",
	}
	multiErr := errortypes.NewMultiError()
	withDates.Validate("filter.timeframe", multiErr)
	assert.True(t, multiErr.HasErrors())

	backwards := desktranscript.Timeframe{
		Preset:   desktranscript.TimeframeCustom,
		FromDate: "2026-03-31",
		ToDate:   "2026-03-01",
	}
	multiErr = errortypes.NewMultiError()
	backwards.Validate("filter.timeframe", multiErr)
	assert.True(t, multiErr.HasErrors())

	unknown := desktranscript.Timeframe{Preset: "Last5Days"}
	multiErr = errortypes.NewMultiError()
	unknown.Validate("filter.timeframe", multiErr)
	assert.True(t, multiErr.HasErrors())
}

func TestConversationFilter_KeyChangesWithEveryNarrowing(t *testing.T) {
	t.Parallel()

	base := desktranscript.ConversationFilter{Timeframe: desktranscript.AllTime()}
	byPerson := base
	byPerson.UserID = pulid.MustNew("usr_")
	byOrigin := base
	byOrigin.Origins = []conversation.ThreadOrigin{conversation.ThreadOriginDesk}

	assert.NotEqual(t, base.Key(), byPerson.Key())
	assert.NotEqual(t, base.Key(), byOrigin.Key())
	assert.NotEqual(t, byPerson.Key(), byOrigin.Key())
}

func TestExport_IsExactlyOneShape(t *testing.T) {
	t.Parallel()

	filter := &desktranscript.ConversationFilter{Timeframe: desktranscript.AllTime()}
	cases := map[string]*desktranscript.Export{
		"a filter export with no filter": {
			RequestedByUserID: pulid.MustNew("usr_"),
			Kind:              desktranscript.ExportKindFilter,
			Status:            desktranscript.ExportStatusPending,
		},
		"a selection with no conversations": {
			RequestedByUserID: pulid.MustNew("usr_"),
			Kind:              desktranscript.ExportKindSelection,
			Status:            desktranscript.ExportStatusPending,
		},
		"more than an export holds": {
			RequestedByUserID: pulid.MustNew("usr_"),
			Kind:              desktranscript.ExportKindFilter,
			Filter:            filter,
			ConversationCount: desktranscript.MaxExportConversations + 1,
			Status:            desktranscript.ExportStatusPending,
		},
	}
	for name, export := range cases {
		multiErr := errortypes.NewMultiError()
		export.Validate(multiErr)
		assert.True(t, multiErr.HasErrors(), name)
	}

	valid := &desktranscript.Export{
		RequestedByUserID: pulid.MustNew("usr_"),
		Kind:              desktranscript.ExportKindFilter,
		Filter:            filter,
		ConversationCount: 3,
		Status:            desktranscript.ExportStatusPending,
	}
	multiErr := errortypes.NewMultiError()
	valid.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestExport_IsDownloadableOnlyWhileReadyAndUnexpired(t *testing.T) {
	t.Parallel()

	expires := int64(1_000)
	export := &desktranscript.Export{
		Status:            desktranscript.ExportStatusSucceeded,
		ArtifactKey:       "desk-transcript-exports/org_1/dtex_1.zip",
		ArtifactExpiresAt: &expires,
	}

	assert.True(t, export.Downloadable(999))
	assert.False(t, export.Downloadable(1_000))
	export.Status = desktranscript.ExportStatusRunning
	assert.False(t, export.Downloadable(999))
}
