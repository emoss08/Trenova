package assistantcompaction

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationreplay"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newCompactionService(
	definition *agentdefinition.Definition,
) (*Service, *assistanttest.StubConversations) {
	conversations := &assistanttest.StubConversations{Thread: &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Dispatch",
	}}

	return &Service{
		logger:        zap.NewNop(),
		conversations: conversations,
		definitions:   &assistanttest.StubDefinitions{Definition: definition},
		permissions:   &agenttest.StubPermissions{},
	}, conversations
}

func TestPrepareCompaction_SummarizesEverythingBeforeTheLatestTurns(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	conversations.Messages = assistanttest.LongConversation(12)
	conversations.Thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	actor := assistanttest.Actor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.Thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.NoError(t, err)

	assert.True(t, conversations.LastList.SinceCompaction,
		"the conversation is read as the model reads it, from any earlier summary on")
	// Twelve turns of four messages: the last two turns are kept whole.
	assert.Equal(t, 10*4-1, plan.Through)
	assert.Equal(t, 40, plan.Summarized)
	assert.Greater(t, plan.Before, plan.After)
	assert.Equal(t, 6200, plan.Instructions, "the prompt's measure carries over")
	assert.Equal(t, []string{conversationreplay.KeptRecent}, plan.Kept)

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
	assert.Contains(t, transcript, "Tool shipment_list returned: ")
	assert.Less(t, len(transcript), compactionTranscriptChars, "the stretch is bounded")
}

func TestPrepareCompaction_RefusesAConversationWithNothingToCompact(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	conversations.Messages = assistanttest.LongConversation(turnhistory.CompactionKeepTurns)
	actor := assistanttest.Actor()

	_, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.Thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.Error(t, err)
	assert.True(
		t,
		errortypes.IsBusinessError(err),
		"the person is told there is nothing to compact",
	)
}

/*
The summary is saved as a message of its own, after everything in the thread,
and the conversation is measured again as it will now read: from the summary,
with the latest turns after it. A retry of a save that landed returns the
summary already saved rather than leaving two.
*/
func TestFinishCompaction_SavesTheSummaryOnceAndMeasuresTheConversationAfter(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	conversations.Messages = assistanttest.LongConversation(12)
	conversations.Thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	actor := assistanttest.Actor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.Thread.ID,
		TenantInfo: actor.TenantInfo(),
		Auto:       true,
	}, actor)
	require.NoError(t, err)

	reply := &CompactionReply{
		Summary: "- Customer 0 to 9 each had 40 loads.",
		Model:   "claude-sonnet-4-5",
	}
	result, err := svc.FinishCompaction(t.Context(), plan, reply, actor)
	require.NoError(t, err)

	require.Len(t, conversations.Appended, 1)
	summary := conversations.Appended[0]
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
	require.NotEmpty(t, conversations.Contexts)
	assert.Equal(t, result.Usage, conversations.Contexts[len(conversations.Contexts)-1].Usage)

	saved := summary
	saved.Sequence = len(conversations.Messages)
	conversations.Messages = append(conversations.Messages, saved)
	conversations.Appended = nil

	again, err := svc.FinishCompaction(t.Context(), plan, reply, actor)
	require.NoError(t, err)
	assert.Empty(t, conversations.Appended, "a retried save does not add a second summary")
	assert.Equal(t, reply.Summary, again.Message.Content)
}

func TestFinishCompaction_RefusesAnEmptySummary(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	actor := assistanttest.Actor()

	_, err := svc.FinishCompaction(t.Context(), &CompactionPlan{ThreadID: conversations.Thread.ID},
		&CompactionReply{Summary: "  "}, actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(
		t,
		conversations.Appended,
		"nothing replaces the conversation when nothing came back",
	)
}
