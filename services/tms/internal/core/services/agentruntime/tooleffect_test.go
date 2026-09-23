package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type steppedClock struct {
	*localEffects

	now int64
}

func (c *steppedClock) Now() int64 {
	c.now += 10

	return c.now
}

func effectRuntime() *Service {
	return newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			queryTool("list_customers", nil, nil),
		}},
		&stubActionRegistry{Tools: []serviceports.AgentTool{
			actionTool("create_report", agent.TierPropose, nil),
		}},
		nil,
	)
}

func TestToolEffect_DerivesFromMetadataAndNamesTheRuntimeTools(t *testing.T) {
	t.Parallel()

	rt := effectRuntime()

	assert.Equal(t, agent.ToolEffectLookup, rt.ToolEffect("list_customers"))
	assert.Equal(t, agent.ToolEffectChange, rt.ToolEffect("create_report"))
	assert.Equal(t, agent.ToolEffectDiscover, rt.ToolEffect(findToolsName))
	assert.Equal(t, agent.ToolEffectAsk, rt.ToolEffect(askUserName))
	assert.Equal(t, agent.ToolEffectPresent, rt.ToolEffect(publishArtifactName))
	assert.Empty(t, rt.ToolEffect("retired_tool"), "a tool that no longer exists has no effect")
}

func TestMarkToolEffects_LabelsCallsAndResultsWithoutTouchingTheStoredCalls(t *testing.T) {
	t.Parallel()

	rt := effectRuntime()
	stored := []conversation.ToolCallRecord{
		{ID: "c1", Name: "list_customers"},
		{ID: "c2", Name: "retired_tool"},
	}
	messages := []conversation.Message{
		{Role: conversation.RoleAssistant, ToolCalls: stored},
		{Role: conversation.RoleTool, ToolName: "list_customers", ToolCallID: "c1"},
		{Role: conversation.RoleTool, ToolName: "retired_tool", ToolCallID: "c2"},
		{Role: conversation.RoleUser, Content: "Thanks."},
	}

	rt.MarkToolEffects(messages)

	assert.Equal(t, agent.ToolEffectLookup, messages[0].ToolCalls[0].Effect)
	assert.Empty(t, messages[0].ToolCalls[1].Effect)
	assert.Equal(t, agent.ToolEffectLookup, messages[1].ToolEffect)
	assert.Empty(t, messages[2].ToolEffect)
	assert.Empty(t, messages[3].ToolEffect)
	assert.Empty(t, stored[0].Effect, "the slice the messages were read with is not rewritten")

	var nilRuntime *Service
	assert.NotPanics(t, func() { nilRuntime.MarkToolEffects(messages) })
}

func TestSummarizeResult(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		tool     string
		document any
		want     string
	}{
		{
			name: "one match names it",
			tool: "list_customers",
			document: map[string]any{
				"count": float64(1),
				"items": []any{map[string]any{"name": "Peak Distributing"}},
			},
			want: "Peak Distributing",
		},
		{
			name: "several matches are counted",
			tool: "list_customers",
			document: map[string]any{
				"count": float64(3),
				"items": []any{map[string]any{"name": "A"}, map[string]any{}, map[string]any{}},
			},
			want: "3 customers",
		},
		{
			name: "a full page says there are more",
			tool: "search_shipments",
			document: map[string]any{
				"count":   float64(25),
				"items":   []any{map[string]any{}},
				"hasMore": true,
			},
			want: "25+ shipments",
		},
		{
			name:     "nothing matched",
			tool:     "list_customers",
			document: map[string]any{"count": float64(0), "items": []any{}},
			want:     "No customers",
		},
		{
			name:     "a noun that is not a plural reads as results",
			tool:     "list_time_off",
			document: []any{map[string]any{}, map[string]any{}},
			want:     "2 results",
		},
		{
			name:     "a record is named",
			tool:     "get_worker",
			document: map[string]any{"firstName": "Maria", "lastName": "Ortiz"},
			want:     "Maria Ortiz",
		},
		{
			name:     "a shipment is named by its number",
			tool:     "get_shipment",
			document: map[string]any{"proNumber": "PRO-1042", "status": "New"},
			want:     "PRO-1042",
		},
		{
			name:     "a page opened is named by its destination",
			tool:     "open_page",
			document: map[string]any{"path": "/reports", "name": "Report library"},
			want:     "Report library",
		},
		{
			name: "a finished report run carries its row count",
			tool: "run_report",
			document: map[string]any{
				"reportName": "Late loads",
				"finished":   true,
				"rowCount":   float64(42),
			},
			want: "Late loads · 42 rows",
		},
		{
			name:     "a queued report run is named alone",
			tool:     "run_report",
			document: map[string]any{"reportName": "Late loads", "finished": false},
			want:     "Late loads",
		},
		{
			name:     "a result with nothing to name has no summary",
			tool:     "get_dispatch_board",
			document: map[string]any{"columns": []any{}},
			want:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, summarizeResult(tc.tool, tc.document))
		})
	}
}

func TestSummarizeOutcome_FailedCallsHaveNone(t *testing.T) {
	t.Parallel()

	call := serviceports.ToolCall{Name: "create_report", Arguments: map[string]any{"name": "Late loads"}}

	assert.Empty(t, summarizeOutcome(call, toolOutcome{failed: true, summary: "Late loads"}))
	assert.Equal(t, "Late loads", summarizeOutcome(call, toolOutcome{
		action: &serviceports.PendingAction{ToolName: "create_report"},
	}))
	assert.Empty(t, summarizeOutcome(call, toolOutcome{content: "done"}))
}

func TestSummaryLine_KeepsOneShortLine(t *testing.T) {
	t.Parallel()

	long := "Quarterly\nlate   loads by customer and lane, with every exception the " +
		"desk raised this year, every carrier it touched and every note it left behind"

	line := summaryLine(long)

	assert.NotContains(t, line, "\n")
	assert.NotContains(t, line, "  ")
	assert.LessOrEqual(t, len([]rune(line)), maxToolSummaryRunes+1)
	assert.True(t, strings.HasSuffix(line, "…"))
}

func TestDrive_StampsEachMessageWhenItWasProducedAndLabelsTheToolTraffic(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("list_customers", map[string]any{"query": "peak"}),
		textTurn("Peak Distributing is on file."),
	}}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			queryTool("list_customers", map[string]any{
				"count": 1,
				"items": []map[string]any{{"name": "Peak Distributing"}},
			}, nil),
		}},
		&stubActionRegistry{},
		nil,
	)

	var events []serviceports.StreamEvent
	req := &serviceports.RunRequest{
		Definition: testDefinition("list_customers"),
		Actor:      testActor(),
		Input:      "Is Peak on file?",
	}
	clock := &steppedClock{
		localEffects: &localEffects{
			s:    rt,
			ctx:  t.Context(),
			emit: func(event serviceports.StreamEvent) { events = append(events, event) },
		},
		now: 1_000,
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), req), clock)
	require.NoError(t, err)

	require.Len(t, result.Messages, 4)
	question, asked, answered, reply := result.Messages[0], result.Messages[1],
		result.Messages[2], result.Messages[3]

	assert.Positive(t, question.CreatedAt, "the question is stamped when the turn opens")
	assert.Equal(t, int64(1_010), asked.CreatedAt)
	assert.Equal(t, int64(1_020), answered.CreatedAt)
	assert.Equal(t, int64(1_030), reply.CreatedAt)

	assert.Equal(t, agent.ToolEffectLookup, answered.ToolEffect)
	assert.Equal(t, "Peak Distributing", answered.ToolSummary)
	assert.Empty(t, asked.ToolCalls[0].Effect, "the call is stored as the model sent it")

	var started *serviceports.AssistantToolStartedEvent
	var finished *serviceports.AssistantToolFinishedEvent
	var message *serviceports.AssistantMessageEvent
	for _, event := range events {
		switch data := event.Data.(type) {
		case serviceports.AssistantToolStartedEvent:
			started = &data
		case serviceports.AssistantToolFinishedEvent:
			finished = &data
		case serviceports.AssistantMessageEvent:
			message = &data
		}
	}

	require.NotNil(t, started)
	require.NotNil(t, finished)
	require.NotNil(t, message)
	assert.Equal(t, agent.ToolEffectLookup, started.Effect)
	assert.Equal(t, agent.ToolEffectLookup, finished.Effect)
	assert.Equal(t, "Peak Distributing", finished.Summary)
	require.Len(t, message.ToolCalls, 1)
	assert.Equal(t, agent.ToolEffectLookup, message.ToolCalls[0].Effect)
}

func TestRun_AProposedWriteIsLabelledAChangeAndNamedFromItsArguments(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_report", map[string]any{"name": "Late loads"}),
		textTurn("I proposed the report."),
	}}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{
			actionTool("create_report", agent.TierPropose, nil),
		}},
		nil,
	)

	var finished *serviceports.AssistantToolFinishedEvent
	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("create_report"),
		Actor:      testActor(),
		Input:      "Save a late loads report.",
		Emit: func(event serviceports.StreamEvent) {
			if data, ok := event.Data.(serviceports.AssistantToolFinishedEvent); ok {
				finished = &data
			}
		},
	})
	require.NoError(t, err)

	require.NotNil(t, finished)
	assert.True(t, finished.Proposed)
	assert.Equal(t, agent.ToolEffectChange, finished.Effect)
	assert.Equal(t, "Late loads", finished.Summary)

	require.Len(t, result.Messages, 4)
	assert.Equal(t, agent.ToolEffectChange, result.Messages[2].ToolEffect)
	assert.Equal(t, "Late loads", result.Messages[2].ToolSummary)
}
