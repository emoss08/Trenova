package deskbench

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func turnWithCalls(tools []string, calls ...CallSummary) *TurnRecord {
	turn := &TurnRecord{Calls: calls}
	for _, name := range tools {
		turn.Tools = append(turn.Tools, &ToolCallRecord{Name: name})
	}

	return turn
}

func TestMeasureCase_CountsTheCacheReadOfEveryCallButATurnsFirst(t *testing.T) {
	t.Parallel()

	result := &CaseResult{
		StartedAt:  time.Unix(0, 0),
		FinishedAt: time.Unix(10, 0),
		Steps: []*StepResult{{Turns: []*TurnRecord{
			turnWithCalls([]string{findToolsName, "list_trailers"},
				CallSummary{Kind: ModelCallChat, InputTokens: 900, CacheWrite: 9000},
				CallSummary{Kind: ModelCallChat, InputTokens: 100, CacheRead: 9900},
				CallSummary{Kind: ModelCallChat, Delegate: "call_d", InputTokens: 5000},
			),
			turnWithCalls([]string{"get_shipment"},
				CallSummary{Kind: ModelCallChat, InputTokens: 50, CacheRead: 9950},
				CallSummary{Kind: ModelCallChat, InputTokens: 50, CacheRead: 9950},
			),
		}}},
	}

	metrics := measureCase(result)

	assert.Equal(t, 10000+10000, metrics.LaterPrompt,
		"a turn's first call writes the cache, and a delegate's calls are its own")
	assert.Equal(t, 9900+9950, metrics.LaterCacheRead)
	assert.Equal(t, 2, metrics.ToolTurns)
	assert.Equal(t, 1, metrics.FindFirstTurns)
}

func TestOpeningTools_LeavesOutWhatIsDeclaredDeferred(t *testing.T) {
	t.Parallel()

	turn := &TurnRecord{ModelCalls: []*ModelCall{{Chat: &agentports.ChatCompletionRequest{
		Tools: []kernelports.ToolSpec{
			{Name: "find_tools"},
			{Name: "get_shipment"},
			{Name: "list_trailers", Deferred: true},
		},
	}}}}

	assert.Equal(t, []string{"find_tools", "get_shipment"}, openingTools(turn))
}

func TestCompareCells_LinesUpEachModelAgainstTheEarlierRun(t *testing.T) {
	t.Parallel()

	share := func(value float64) *float64 { return &value }
	before := &Report{Summary: Summary{Cells: []CellRow{{
		Provider: "claude-haiku-5-5@low",
		PassK:    Stat{Mean: 0.6, N: 4},
		Measures: Measures{
			ModelCalls:      Stat{Mean: 6, N: 4},
			LaterCacheShare: share(0.31),
			FindFirstShare:  share(0.4),
		},
	}}}}
	after := &Report{Summary: Summary{Cells: []CellRow{
		{
			Provider: "claude-haiku-5-5@low",
			PassK:    Stat{Mean: 0.65, N: 4},
			Measures: Measures{
				ModelCalls:      Stat{Mean: 4.5, N: 4},
				LaterCacheShare: share(0.92),
				FindFirstShare:  share(0.1),
			},
		},
		{Provider: "new-cell"},
	}}}

	changes := CompareCells(before, after)

	require.Len(t, changes, 1, "a cell the earlier run did not have is not compared")
	var out strings.Builder
	renderCellChanges(&out, changes)
	rendered := out.String()
	assert.Contains(t, rendered, "CACHED, CALLS 2+")
	assert.Contains(t, rendered, "31% → 92%")
	assert.Contains(t, rendered, "40% → 10%")
}
