package benchresult

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
)

func turnWithCalls(tools []string, calls ...benchrecord.CallSummary) *benchrecord.TurnRecord {
	turn := &benchrecord.TurnRecord{Calls: calls}
	for _, name := range tools {
		turn.Tools = append(turn.Tools, &benchrecord.ToolCallRecord{Name: name})
	}

	return turn
}

func TestMeasureCase_CountsTheCacheReadOfEveryCallButATurnsFirst(t *testing.T) {
	t.Parallel()

	result := &CaseResult{
		StartedAt:  time.Unix(0, 0),
		FinishedAt: time.Unix(10, 0),
		Steps: []*StepResult{{Turns: []*benchrecord.TurnRecord{
			turnWithCalls(
				[]string{findToolsName, "equipment_list_trailers"},
				benchrecord.CallSummary{
					Kind:        benchrecord.ModelCallChat,
					InputTokens: 900,
					CacheWrite:  9000,
				},
				benchrecord.CallSummary{
					Kind:        benchrecord.ModelCallChat,
					InputTokens: 100,
					CacheRead:   9900,
				},
				benchrecord.CallSummary{
					Kind:        benchrecord.ModelCallChat,
					Delegate:    "call_d",
					InputTokens: 5000,
				},
			),
			turnWithCalls(
				[]string{"shipment_get"},
				benchrecord.CallSummary{
					Kind:        benchrecord.ModelCallChat,
					InputTokens: 50,
					CacheRead:   9950,
				},
				benchrecord.CallSummary{
					Kind:        benchrecord.ModelCallChat,
					InputTokens: 50,
					CacheRead:   9950,
				},
			),
		}}},
	}

	metrics := MeasureCase(result)

	assert.Equal(t, 10000+10000, metrics.LaterPrompt,
		"a turn's first call writes the cache, and a delegate's calls are its own")
	assert.Equal(t, 9900+9950, metrics.LaterCacheRead)
	assert.Equal(t, 2, metrics.ToolTurns)
	assert.Equal(t, 1, metrics.FindFirstTurns)
}

func TestOpeningTools_LeavesOutWhatIsDeclaredDeferred(t *testing.T) {
	t.Parallel()

	turn := &benchrecord.TurnRecord{
		ModelCalls: []*benchrecord.ModelCall{{Chat: &llmports.ChatCompletionRequest{
			Tools: []kernelports.ToolSpec{
				{Name: "find_tools"},
				{Name: "shipment_get"},
				{Name: "equipment_list_trailers", Deferred: true},
			},
		}}},
	}

	assert.Equal(t, []string{"find_tools", "shipment_get"}, openingTools(turn))
}
