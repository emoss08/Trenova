package assistantservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/contextusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A saved turn is measured, the measure kept on the thread, and the reader told on
the turn's stream, so the composer's meter moves with the reply.
*/
func TestMeasureAfterTurn_KeepsTheMeasureAndTellsTheReader(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(
		&assistanttest.ScriptedCompletion{},
		assistanttest.Definition(),
	)
	conversations.Messages = assistanttest.LongConversation(3)
	actor := assistanttest.Actor()

	plan := &TurnPlan{Turn: agentruntime.TurnState{System: strings.Repeat("s", 8000)}}
	saved := []conversation.Message{{Role: conversation.RoleAssistant, Model: "gpt-4o"}}

	var events []kernelports.StreamEvent
	svc.measureAfterTurn(t.Context(), conversations.Thread, plan, saved, 0, actor.TenantInfo(),
		func(event kernelports.StreamEvent) { events = append(events, event) })

	require.NotNil(t, conversations.Thread.ContextUsage)
	usage := conversations.Thread.ContextUsage
	assert.Equal(t, 2000, usage.Instructions)
	assert.Equal(t, 128_000, usage.Window, "measured against the model that answered")
	require.Len(t, conversations.Contexts, 1)

	require.Len(t, events, 1)
	assert.Equal(t, assistantstreamports.AssistantEventContext, events[0].Event)
	data, ok := events[0].Data.(assistantstreamports.AssistantContextEvent)
	require.True(t, ok)
	assert.Equal(t, conversations.Thread.ID, data.ThreadID)
	assert.Equal(t, usage.Total(), data.Usage.Total())
}

/*
A turn answered by a provider configured with its own window is measured
against that window, and a later measure taken without a reply, such as the
one a compaction plans from, keeps it rather than reading the model id again.
*/
func TestMeasureAfterTurn_KeepsTheProvidersConfiguredWindow(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(
		&assistanttest.ScriptedCompletion{},
		assistanttest.Definition(),
	)
	conversations.Messages = assistanttest.LongConversation(3)
	actor := assistanttest.Actor()

	plan := &TurnPlan{Turn: agentruntime.TurnState{System: "Help dispatch."}}
	saved := []conversation.Message{{Role: conversation.RoleAssistant, Model: "qwen2.5-coder:32b"}}

	svc.measureAfterTurn(
		t.Context(),
		conversations.Thread,
		plan,
		saved,
		131_072,
		actor.TenantInfo(),
		nil,
	)

	require.NotNil(t, conversations.Thread.ContextUsage)
	assert.Equal(t, 131_072, conversations.Thread.ContextUsage.Window,
		"the configured window wins over the 32k the id names")

	later := contextusage.CurrentUsage(conversations.Thread, conversations.Messages, nil)
	assert.Equal(t, 131_072, later.Window)
	assert.Equal(t, "qwen2.5-coder:32b", later.Model)
}
