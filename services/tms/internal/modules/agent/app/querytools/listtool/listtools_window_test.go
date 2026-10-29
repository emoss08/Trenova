package listtool

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// "Expiring in the next 30 days" used to start at the current second, so a
// credential that expired earlier today was not expiring in the next 30 days.
func TestListTool_NextNDaysIncludesTheRestOfToday(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := New(probeSpec(capture))

	_, err := tool.Query(t.Context(), querytooltest.TestParams(filterParams(
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
	assert.Zero(t, lower%filtercatalog.SecondsPerDay, "and on a day boundary")
	assert.Greater(t, now-lower, int64(0))
	assert.Less(t, now-lower, int64(filtercatalog.SecondsPerDay), "that boundary is today's")
	assert.Equal(t, lower+31*filtercatalog.SecondsPerDay-1, upper, "through the end of day thirty")
}

func TestListTool_LastNDaysOpensOnADayBoundary(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := New(probeSpec(capture))

	_, err := tool.Query(t.Context(), querytooltest.TestParams(filterParams(
		map[string]any{"field": "expiresAt", "operator": "lastndays", "days": "7"},
	)))
	require.NoError(t, err, "a number sent as a string is still a number")

	lower, ok := capture.opts.FieldFilters[0].Value.(int64)
	require.True(t, ok)
	assert.Zero(t, lower%filtercatalog.SecondsPerDay)
	assert.Less(t, timeutils.NowUnix()-lower, int64(8*filtercatalog.SecondsPerDay))
}

// The window opens on the organization's midnight, not Greenwich's. For a
// carrier in New York that is a four- or five-hour difference, which at the
// edge is a whole day of credentials.
func TestListTool_WindowOpensOnTheOrganizationsMidnight(t *testing.T) {
	t.Parallel()

	capture := &capturedList{}
	tool := New(probeSpec(capture))

	_, err := tool.Query(t.Context(), querytooltest.TestParamsIn("America/New_York", filterParams(
		map[string]any{"field": "expiresAt", "operator": "nextndays", "days": 30},
	)))
	require.NoError(t, err)

	lower, ok := capture.opts.FieldFilters[0].Value.(int64)
	require.True(t, ok)
	expected, err := timeutils.DayStartUnix(timeutils.NowUnix(), "America/New_York")
	require.NoError(t, err)
	assert.Equal(t, expected, lower)
}
