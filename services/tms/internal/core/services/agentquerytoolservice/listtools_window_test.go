package agentquerytoolservice

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// "Expiring in the next 30 days" used to start at the current second, so a
// credential that expired earlier today was not expiring in the next 30 days.
func TestListTool_NextNDaysIncludesTheRestOfToday(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "nextndays", "days": 30},
	)))
	require.NoError(t, err)
	require.Len(t, capture.opts.FieldFilters, 2)

	now := timeutils.NowUnix()
	lower, ok := capture.opts.FieldFilters[0].Value.(int64)
	require.True(t, ok)
	upper, ok := capture.opts.FieldFilters[1].Value.(int64)
	require.True(t, ok)

	assert.Equal(t, dbtype.OpGreaterThanOrEqual, capture.opts.FieldFilters[0].Operator)
	assert.LessOrEqual(t, lower, now, "the window opens no later than now")
	assert.Zero(t, lower%secondsPerDay, "and on a day boundary")
	assert.Greater(t, now-lower, int64(0))
	assert.Less(t, now-lower, int64(secondsPerDay), "that boundary is today's")
	assert.Equal(t, lower+31*secondsPerDay-1, upper, "through the end of day thirty")
}

func TestListTool_LastNDaysOpensOnADayBoundary(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "lastndays", "days": "7"},
	)))
	require.NoError(t, err, "a number sent as a string is still a number")

	lower, ok := capture.opts.FieldFilters[0].Value.(int64)
	require.True(t, ok)
	assert.Zero(t, lower%secondsPerDay)
	assert.Less(t, timeutils.NowUnix()-lower, int64(8*secondsPerDay))
}

func testParamsIn(timezone string, params map[string]any) serviceports.QueryToolParams {
	out := testParams(params)
	out.Timezone = timezone

	return out
}

// The window opens on the organization's midnight, not Greenwich's. For a
// carrier in New York that is a four- or five-hour difference, which at the
// edge is a whole day of credentials.
func TestListTool_WindowOpensOnTheOrganizationsMidnight(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := newListTool(probeSpec(capture))

	_, err := tool.Query(t.Context(), testParamsIn("America/New_York", filterParams(
		map[string]any{"field": "expiresAt", "operator": "nextndays", "days": 30},
	)))
	require.NoError(t, err)

	lower, ok := capture.opts.FieldFilters[0].Value.(int64)
	require.True(t, ok)
	expected, err := timeutils.DayStartUnix(timeutils.NowUnix(), "America/New_York")
	require.NoError(t, err)
	assert.Equal(t, expected, lower)
}

// "today" is the organization's today, and a bare date is midnight there.
func TestListTool_ReadsNamedAndCalendarDaysInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	ny := clock{now: lateEveningNewYork, timezone: "America/New_York"}

	today, ok := namedDay("today", ny)
	require.True(t, ok)
	assert.Equal(t, int64(1789790400), today, "still the 19th in New York")

	seconds, err := coerceDateValue("expiresAt", "2026-09-19", ny)
	require.NoError(t, err)
	assert.Equal(t, int64(1789790400), seconds)
}
