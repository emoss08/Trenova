package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("customer_list", nil, nil),
		}},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{
			actionTool("report_create", agent.TierPropose, nil),
		}},
		nil,
	)
}

func TestToolEffect_DerivesFromMetadataAndNamesTheRuntimeTools(t *testing.T) {
	t.Parallel()

	rt := effectRuntime()

	assert.Equal(t, agenttooltrust.ToolEffectLookup, rt.access.ToolEffect("customer_list"))
	assert.Equal(t, agenttooltrust.ToolEffectChange, rt.access.ToolEffect("report_create"))
	assert.Equal(
		t,
		agenttooltrust.ToolEffectDiscover,
		rt.access.ToolEffect(runtimetools.FindToolsName),
	)
	assert.Equal(t, agenttooltrust.ToolEffectAsk, rt.access.ToolEffect(askuser.Name))
	assert.Equal(
		t,
		agenttooltrust.ToolEffectPresent,
		rt.access.ToolEffect(runtimetools.PublishArtifactName),
	)
	assert.Empty(
		t,
		rt.access.ToolEffect("retired_tool"),
		"a tool that no longer exists has no effect",
	)
}

func TestMarkToolEffects_LabelsCallsAndResultsWithoutTouchingTheStoredCalls(t *testing.T) {
	t.Parallel()

	rt := effectRuntime()
	stored := []conversation.ToolCallRecord{
		{ID: "c1", Name: "customer_list"},
		{ID: "c2", Name: "retired_tool"},
	}
	messages := []conversation.Message{
		{Role: conversation.RoleAssistant, ToolCalls: stored},
		{Role: conversation.RoleTool, ToolName: "customer_list", ToolCallID: "c1"},
		{Role: conversation.RoleTool, ToolName: "retired_tool", ToolCallID: "c2"},
		{Role: conversation.RoleUser, Content: "Thanks."},
	}

	rt.access.MarkToolEffects(messages)

	assert.Equal(t, agenttooltrust.ToolEffectLookup, messages[0].ToolCalls[0].Effect)
	assert.Empty(t, messages[0].ToolCalls[1].Effect)
	assert.Equal(t, agenttooltrust.ToolEffectLookup, messages[1].ToolEffect)
	assert.Empty(t, messages[2].ToolEffect)
	assert.Empty(t, messages[3].ToolEffect)
	assert.Empty(t, stored[0].Effect, "the slice the messages were read with is not rewritten")

	var nilRuntime *Service
	assert.NotPanics(t, func() { nilRuntime.access.MarkToolEffects(messages) })
}

func TestDrive_StampsEachMessageWhenItWasProducedAndLabelsTheToolTraffic(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("customer_list", map[string]any{"query": "peak"}),
		runtimetest.TextTurn("Peak Distributing is on file."),
	}}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("customer_list", map[string]any{
				"count": 1,
				"items": []map[string]any{{"name": "Peak Distributing"}},
			}, nil),
		}},
		&stubActionRegistry{},
		nil,
	)

	var events []kernelports.StreamEvent
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("customer_list"),
		Actor:      runtimetest.Actor(),
		Input:      "Is Peak on file?",
	}
	clock := &steppedClock{
		localEffects: &localEffects{
			s:    rt,
			ctx:  t.Context(),
			emit: func(event kernelports.StreamEvent) { events = append(events, event) },
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

	assert.Equal(t, agenttooltrust.ToolEffectLookup, answered.ToolEffect)
	assert.Equal(t, "Peak Distributing", answered.ToolSummary)
	assert.Empty(t, asked.ToolCalls[0].Effect, "the call is stored as the model sent it")

	var started *assistantstreamports.AssistantToolStartedEvent
	var finished *assistantstreamports.AssistantToolFinishedEvent
	var message *assistantstreamports.AssistantMessageEvent
	for _, event := range events {
		switch data := event.Data.(type) {
		case assistantstreamports.AssistantToolStartedEvent:
			started = &data
		case assistantstreamports.AssistantToolFinishedEvent:
			finished = &data
		case assistantstreamports.AssistantMessageEvent:
			message = &data
		}
	}

	require.NotNil(t, started)
	require.NotNil(t, finished)
	require.NotNil(t, message)
	assert.Equal(t, agenttooltrust.ToolEffectLookup, started.Effect)
	assert.Equal(t, agenttooltrust.ToolEffectLookup, finished.Effect)
	assert.Equal(t, "Peak Distributing", finished.Summary)
	require.Len(t, message.ToolCalls, 1)
	assert.Equal(t, agenttooltrust.ToolEffectLookup, message.ToolCalls[0].Effect)
}

func TestRun_AProposedWriteIsLabelledAChangeAndNamedFromItsArguments(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_create", map[string]any{"name": "Late loads"}),
		runtimetest.TextTurn("I proposed the report."),
	}}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{
			actionTool("report_create", agent.TierPropose, nil),
		}},
		nil,
	)

	var finished *assistantstreamports.AssistantToolFinishedEvent
	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_create"),
		Actor:      runtimetest.Actor(),
		Input:      "Save a late loads report.",
		Emit: func(event kernelports.StreamEvent) {
			if data, ok := event.Data.(assistantstreamports.AssistantToolFinishedEvent); ok {
				finished = &data
			}
		},
	})
	require.NoError(t, err)

	require.NotNil(t, finished)
	assert.True(t, finished.Proposed)
	assert.Equal(t, agenttooltrust.ToolEffectChange, finished.Effect)
	assert.Equal(t, "Late loads", finished.Summary)

	require.Len(t, result.Messages, 4)
	assert.Equal(t, agenttooltrust.ToolEffectChange, result.Messages[2].ToolEffect)
	assert.Equal(t, "Late loads", result.Messages[2].ToolSummary)
}
