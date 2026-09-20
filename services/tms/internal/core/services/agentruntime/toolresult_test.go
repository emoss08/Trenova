package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The clock from the transcript this file exists because of: 2026-09-19.
const transcriptNow int64 = 1789776000

/*
The failure this fixes, with the numbers it actually failed on.

Asked which drivers had a medical card expiring within thirty days, the agent
was handed medicalCardExpiry: 1791591001. Having no arithmetic, it reasoned
"56 years × 365.25 days", decided the date was July 2026, called it already
past, and told a dispatcher nobody was expiring. The card expires 2026-10-10 —
twenty-one days out, squarely inside the window it was asked about. A driver
whose card lapses in three weeks kept getting dispatched.

No prompt fixes this. The model is being asked to do something it cannot do.
The tool has a clock and can simply answer.
*/
func TestHumanizeDates_RendersTheDateThatWasMisread(t *testing.T) {
	t.Parallel()

	document := map[string]any{"medicalCardExpiry": float64(1791591001)}

	humanized, ok := humanizeDates(document, transcriptNow).(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "2026-10-10 (in 21 days)", humanized["medicalCardExpiry"],
		"the model read this as July 2026 and reported the opposite of the truth")
}

func TestHumanizeDates_RendersADistantDateThatWasAlsoMisread(t *testing.T) {
	t.Parallel()

	// Read as "March 22, 2027" in the transcript. It is October 2027.
	document := map[string]any{"medicalCardExpiry": float64(1824423001)}

	humanized, _ := humanizeDates(document, transcriptNow).(map[string]any)
	assert.Equal(t, "2027-10-25 (in 401 days)", humanized["medicalCardExpiry"])
}

func TestHumanizeDates_NamesTheNearDaysRatherThanCountingThem(t *testing.T) {
	t.Parallel()

	const day = 86400
	for _, tc := range []struct {
		name     string
		ts       int64
		expected string
	}{
		{"today", transcriptNow, "2026-09-19 (today)"},
		{"tomorrow", transcriptNow + day, "2026-09-20 (tomorrow)"},
		{"yesterday", transcriptNow - day, "2026-09-18 (yesterday)"},
		{"lapsed", transcriptNow - 42*day, "2026-08-08 (42 days ago)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			humanized, _ := humanizeDates(
				map[string]any{"expiresAt": float64(tc.ts)}, transcriptNow,
			).(map[string]any)

			assert.Equal(t, tc.expected, humanized["expiresAt"])
		})
	}
}

/*
A count is not an instant.

daysUntilExpiry is already shipped by list_expiring_credentials and its name
ends in "expiry", so the name test alone would rewrite 21 into a date in 1970.
The range check is what stops it, and it is the reason both tests are applied
rather than either one.
*/
func TestHumanizeDates_LeavesACountAlone(t *testing.T) {
	t.Parallel()

	humanized, _ := humanizeDates(map[string]any{
		"daysUntilExpiry": float64(21),
		"expiresAt":       float64(1791591001),
	}, transcriptNow).(map[string]any)

	assert.InDelta(t, 21.0, humanized["daysUntilExpiry"], 0)
	assert.Equal(t, "2026-10-10 (in 21 days)", humanized["expiresAt"])
}

// A money figure can land inside the epoch range. Only a date-like name gets
// rewritten, so an invoice total stays a number it can be added up from.
func TestHumanizeDates_LeavesAFigureInEpochRangeAlone(t *testing.T) {
	t.Parallel()

	humanized, _ := humanizeDates(map[string]any{
		"totalAmount": float64(1791591001),
		"dueDate":     float64(1791591001),
	}, transcriptNow).(map[string]any)

	assert.InDelta(t, 1791591001.0, humanized["totalAmount"], 0)
	assert.Equal(t, "2026-10-10 (in 21 days)", humanized["dueDate"])
}

// "reason" ends in "on", which is why that suffix is not in the set.
func TestHumanizeDates_DoesNotMatchAWordThatMerelyEndsLikeOne(t *testing.T) {
	t.Parallel()

	humanized, _ := humanizeDates(map[string]any{
		"reason": "Three drivers already off that week.",
		"season": float64(1791591001),
	}, transcriptNow).(map[string]any)

	assert.Equal(t, "Three drivers already off that week.", humanized["reason"])
	assert.InDelta(t, 1791591001.0, humanized["season"], 0)
}

// A zero is how the row types spell "not set". Rendering it as 1970 would
// invent a lapsed credential for every driver who has not filed one.
func TestHumanizeDates_LeavesAnUnsetDateAlone(t *testing.T) {
	t.Parallel()

	humanized, _ := humanizeDates(map[string]any{
		"medicalCardExpiry": float64(0),
		"terminationDate":   nil,
	}, transcriptNow).(map[string]any)

	assert.InDelta(t, 0.0, humanized["medicalCardExpiry"], 0)
	assert.Nil(t, humanized["terminationDate"])
}

// The rows arrive nested inside a search outcome, which is the only shape that
// matters in production.
func TestHumanizeDates_ReachesRowsNestedInAResult(t *testing.T) {
	t.Parallel()

	humanized, _ := humanizeDates(map[string]any{
		"count": float64(1),
		"items": []any{
			map[string]any{
				"workerName": "Mike Johnson",
				"profile":    map[string]any{"medicalCardExpiry": float64(1791591001)},
			},
		},
	}, transcriptNow).(map[string]any)

	items, ok := humanized["items"].([]any)
	require.True(t, ok)
	row, ok := items[0].(map[string]any)
	require.True(t, ok)
	profile, ok := row["profile"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "2026-10-10 (in 21 days)", profile["medicalCardExpiry"])
}

// The whole point is that a tool written later inherits this without knowing it
// exists, so the rule has to live on the encode path rather than in the tools.
func TestEncodeToolResult_HumanizesWhateverATypedToolReturned(t *testing.T) {
	t.Parallel()

	type row struct {
		Name      string `json:"name"`
		ExpiresAt int64  `json:"expiresAt"`
	}

	encoded, err := encodeToolResult([]row{{Name: "Mike Johnson", ExpiresAt: 1791591001}},
		transcriptNow)
	require.NoError(t, err)

	assert.Contains(t, encoded, "2026-10-10 (in 21 days)")
	assert.NotContains(t, encoded, "1791591001",
		"the epoch is replaced, not supplemented: nothing downstream can use it")
}

// The rendered prefix is the form the date filters already take, so a date read
// out of one result can be sent straight back into the next call.
func TestEncodeToolResult_ProducesADatePrefixTheFiltersAccept(t *testing.T) {
	t.Parallel()

	encoded, err := encodeToolResult(
		map[string]any{"expiresAt": 1791591001}, transcriptNow)
	require.NoError(t, err)

	var decoded map[string]string
	require.NoError(t, sonic.Unmarshal([]byte(encoded), &decoded))

	assert.Equal(t, "2026-10-10", strings.SplitN(decoded["expiresAt"], " ", 2)[0])
}
