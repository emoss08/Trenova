package assistantservice

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// longConversation is a conversation of turns that each read a long listing,
// numbered as the thread stores them.
func longConversation(turns int) []conversation.Message {
	history := make([]conversation.Message, 0, turns*4)
	for turn := range turns {
		callID := fmt.Sprintf("call_%d", turn)
		history = append(history,
			conversation.Message{Role: conversation.RoleUser, Content: fmt.Sprintf("Which loads for customer %d?", turn)},
			conversation.Message{
				Role:      conversation.RoleAssistant,
				ToolCalls: []conversation.ToolCallRecord{{ID: callID, Name: "list_shipments"}},
			},
			conversation.Message{
				Role:       conversation.RoleTool,
				ToolCallID: callID,
				ToolName:   "list_shipments",
				Content:    strings.Repeat(`{"pro":"S-1001","status":"InTransit"},`, 800),
			},
			conversation.Message{
				Role:    conversation.RoleAssistant,
				Content: fmt.Sprintf("Customer %d has 40 loads. ", turn) + strings.Repeat("Most are on time. ", 120),
			},
		)
	}
	for idx := range history {
		history[idx].Sequence = idx
		history[idx].Kind = conversation.MessageKindMessage
	}

	return history
}

func TestPrepareCompaction_SummarizesEverythingBeforeTheLatestTurns(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(12)
	conversations.thread.ContextUsage = &conversation.ContextUsage{Instructions: 6200, Model: "claude-sonnet-4-5"}
	actor := testActor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.NoError(t, err)

	assert.True(t, conversations.lastList.SinceCompaction,
		"the conversation is read as the model reads it, from any earlier summary on")
	// Twelve turns of four messages: the last two turns are kept whole.
	assert.Equal(t, 10*4-1, plan.Through)
	assert.Equal(t, 40, plan.Summarized)
	assert.Greater(t, plan.Before, plan.After)
	assert.Equal(t, 6200, plan.Instructions, "the prompt's measure carries over")
	assert.Equal(t, []string{conversation.KeptRecent}, plan.Kept)

	request := plan.Request
	require.NotNil(t, request)
	assert.Equal(t, conversation.SummaryTokenBudget, request.MaxTokens)
	assert.Contains(t, request.System, "pinned facts",
		"the summarizer is told the pinned facts are kept apart and not to restate them")
	require.Len(t, request.Messages, 1)
	transcript := request.Messages[0].Content
	assert.Contains(t, transcript, "Person: Which loads for customer 0?")
	assert.Contains(t, transcript, "Agent: Customer 9 has 40 loads.")
	assert.NotContains(t, transcript, "customer 10?", "the kept turns are not summarized")
	assert.Contains(t, transcript, "Tool list_shipments returned: ")
	assert.Less(t, len(transcript), compactionTranscriptChars, "the stretch is bounded")
}

func TestPrepareCompaction_RefusesAConversationWithNothingToCompact(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(agentruntime.CompactionKeepTurns)
	actor := testActor()

	_, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err), "the person is told there is nothing to compact")
}

/*
The summary is saved as a message of its own, after everything in the thread,
and the conversation is measured again as it will now read: from the summary,
with the latest turns after it. A retry of a save that landed returns the
summary already saved rather than leaving two.
*/
func TestFinishCompaction_SavesTheSummaryOnceAndMeasuresTheConversationAfter(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(12)
	conversations.thread.ContextUsage = &conversation.ContextUsage{Instructions: 6200, Model: "claude-sonnet-4-5"}
	actor := testActor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.thread.ID,
		TenantInfo: actor.TenantInfo(),
		Auto:       true,
	}, actor)
	require.NoError(t, err)

	reply := &CompactionReply{Summary: "- Customer 0 to 9 each had 40 loads.", Model: "claude-sonnet-4-5"}
	result, err := svc.FinishCompaction(t.Context(), plan, reply, actor)
	require.NoError(t, err)

	require.Len(t, conversations.appended, 1)
	summary := conversations.appended[0]
	assert.Equal(t, conversation.MessageKindCompaction, summary.Kind)
	assert.Equal(t, conversation.RoleUser, summary.Role)
	assert.Equal(t, reply.Summary, summary.Content)
	require.NotNil(t, summary.Compaction)
	assert.True(t, summary.Compaction.Auto)
	assert.Equal(t, plan.Through, summary.Compaction.Through)
	assert.Equal(t, plan.Before, summary.Compaction.Before)
	assert.Less(t, summary.Compaction.After, plan.Before-conversation.MinCompactionTokens,
		"the compacted stretch gave back its room")

	require.NotNil(t, result.Usage)
	assert.Equal(t, summary.Compaction.After, result.Usage.Total())
	assert.Equal(t, 6200, result.Usage.Instructions, "compaction never touches the instructions")
	require.NotEmpty(t, conversations.contexts)
	assert.Equal(t, result.Usage, conversations.contexts[len(conversations.contexts)-1].Usage)

	saved := summary
	saved.Sequence = len(conversations.messages)
	conversations.messages = append(conversations.messages, saved)
	conversations.appended = nil

	again, err := svc.FinishCompaction(t.Context(), plan, reply, actor)
	require.NoError(t, err)
	assert.Empty(t, conversations.appended, "a retried save does not add a second summary")
	assert.Equal(t, reply.Summary, again.Message.Content)
}

func TestFinishCompaction_RefusesAnEmptySummary(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	actor := testActor()

	_, err := svc.FinishCompaction(t.Context(), &CompactionPlan{ThreadID: conversations.thread.ID},
		&CompactionReply{Summary: "  "}, actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, conversations.appended, "nothing replaces the conversation when nothing came back")
}

/*
A saved turn is measured, the measure kept on the thread, and the reader told on
the turn's stream, so the composer's meter moves with the reply.
*/
func TestMeasureAfterTurn_KeepsTheMeasureAndTellsTheReader(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(3)
	actor := testActor()

	plan := &TurnPlan{Turn: agentruntime.TurnState{System: strings.Repeat("s", 8000)}}
	saved := []conversation.Message{{Role: conversation.RoleAssistant, Model: "gpt-4o"}}

	var events []serviceports.StreamEvent
	svc.measureAfterTurn(t.Context(), conversations.thread, plan, saved, actor.TenantInfo(),
		func(event serviceports.StreamEvent) { events = append(events, event) })

	require.NotNil(t, conversations.thread.ContextUsage)
	usage := conversations.thread.ContextUsage
	assert.Equal(t, 2000, usage.Instructions)
	assert.Equal(t, 128_000, usage.Window, "measured against the model that answered")
	require.Len(t, conversations.contexts, 1)

	require.Len(t, events, 1)
	assert.Equal(t, serviceports.AssistantEventContext, events[0].Event)
	data, ok := events[0].Data.(serviceports.AssistantContextEvent)
	require.True(t, ok)
	assert.Equal(t, conversations.thread.ID, data.ThreadID)
	assert.Equal(t, usage.Total(), data.Usage.Total())
}
