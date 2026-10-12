package agentruntime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stored result is read back for the transcript exactly as the tool
// returned it, close tag and all, and a line that is not a fenced result
// says so rather than yielding an empty payload.
func TestUnfenceToolResult_ReadsBackWhatWasFenced(t *testing.T) {
	t.Parallel()

	payload := `{"note":"see </untrusted_data> in the docs","rows":[1,2]}`
	name, got, ok := UnfenceToolResult(FenceToolResult("list_shipments", payload))

	require.True(t, ok)
	assert.Equal(t, "list_shipments", name)
	assert.Equal(t, payload, got)

	_, _, ok = UnfenceToolResult(`Tool "run_report" failed: no such report`)
	assert.False(t, ok)
	_, _, ok = UnfenceToolResult("")
	assert.False(t, ok)
}

/*
Asked which loads still need a driver, gpt-6-luna was handed the dispatch
board cut at 12,000 characters. Its keys arrive sorted, so the cut kept drivers
and moves and dropped the summary and note that held the count it was asked
for. An oversized object keeps every small field whole and shortens its lists.
*/
func TestTruncateToolResult_KeepsTotalsAndShortensTheLists(t *testing.T) {
	t.Parallel()

	moves := make([]map[string]any, 0, 200)
	for idx := range 200 {
		moves = append(moves, map[string]any{
			"proNumber": fmt.Sprintf("S%010d", idx),
			"origin":    "Los Angeles Terminal in Los Angeles, CA",
			"urgency":   "Late",
		})
	}
	payload, err := sonic.ConfigStd.Marshal(map[string]any{
		"drivers": []map[string]any{{"name": "Robert Brown"}},
		"moves":   moves,
		"note":    "200 moves in the window",
		"summary": map[string]any{"uncoveredMoves": 80, "lateMoves": 78},
	})
	require.NoError(t, err)
	require.Greater(t, len(payload), maxToolResultChars)

	out := truncateToolResult(string(payload))

	body, note, found := strings.Cut(out, "\n\n[")
	require.True(t, found, "the shortening is named")
	assert.Contains(t, note, "moves shows ")
	assert.Contains(t, note, " of 200")
	assert.LessOrEqual(t, len(out), maxToolResultChars)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal([]byte(body), &decoded), "what is kept is whole JSON")
	assert.Equal(t, map[string]any{"uncoveredMoves": float64(80), "lateMoves": float64(78)},
		decoded["summary"])
	assert.Equal(t, "200 moves in the window", decoded["note"])
	assert.NotEmpty(t, decoded["moves"])
	assert.Less(t, len(decoded["moves"].([]any)), 200)
}
