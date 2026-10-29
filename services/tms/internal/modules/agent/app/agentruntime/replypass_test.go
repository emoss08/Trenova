package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func passRun(t *testing.T, reply string, shown *agentruntimeports.ShownArtifact) (
	*agentruntimeports.RunResult,
	*[]kernelports.StreamEvent,
) {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_list", map[string]any{}),
		runtimetest.TextTurn(reply),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		queryTool("shipment_list", map[string]any{"items": []any{}}, nil),
	}}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_list"),
		Actor:      runtimetest.Actor(),
		Input:      "Which loads are late?",
		ToolObserver: func(agentruntimeports.ToolObservation) (*agentruntimeports.ShownArtifact, error) {
			return shown, nil
		},
	}
	events := recordEvents(req)

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return result, events
}

func regroundedActions(events *[]kernelports.StreamEvent) []assistantports.RegroundAction {
	actions := make([]assistantports.RegroundAction, 0, 2)
	for _, event := range *events {
		if data, ok := event.Data.(assistantports.AssistantReplyRegroundedEvent); ok {
			actions = append(actions, data.Action)
		}
	}

	return actions
}

// The recorded reply holds no internal id and no reprint; the corrected reply
// is sent whole in place of the streamed one, with no restart, since the
// model was not asked again, and each pass is in the trajectory.
func TestDrive_PassesOverTheFinalReply(t *testing.T) {
	t.Parallel()

	kept := runtimetest.KeptShipments(25)
	reply := "PRO-1 (ID **shp_01M3Q2Y4SRFE0YW60JY6F5NRW7**) is the oldest.\n" + runtimetest.MarkdownReprint(
		8,
	)

	result, events := passRun(t, reply, &kept)

	assert.NotContains(t, result.Reply, "shp_01M3Q2Y4SRFE0YW60JY6F5NRW7")
	assert.NotContains(t, result.Reply, "| --- |")
	assert.Contains(t, result.Reply, replycheck.ArtifactRef(&kept))
	assert.Equal(t, result.Reply, result.Messages[len(result.Messages)-1].Content)
	assert.Equal(t, []assistantports.RegroundAction{
		assistantports.RegroundStripIDs,
		assistantports.RegroundPointToTable,
	}, regroundedActions(events))

	assertReplacedWithoutRestart(t, *events, result.Reply, replycheck.ReplyPassReplacedReason)
}

// assertReplacedWithoutRestart checks a corrected reply reached the reader as
// one reply_replaced carrying the recorded text, after the last streamed
// piece of the reply, and that nothing in the turn announced a retry.
func assertReplacedWithoutRestart(
	t *testing.T,
	events []kernelports.StreamEvent,
	reply, reason string,
) {
	t.Helper()

	var replaced []assistantports.AssistantReplyReplacedEvent
	afterReplaced := 0
	for _, event := range events {
		assert.NotEqual(t, assistantports.AssistantEventRetrying, event.Event,
			"a correction the model was not asked for is not a retry")
		if data, ok := event.Data.(assistantports.AssistantReplyReplacedEvent); ok {
			assert.Equal(t, assistantports.AssistantEventReplyReplaced, event.Event)
			replaced = append(replaced, data)
			afterReplaced = 0
			continue
		}
		if _, ok := event.Data.(assistantports.AssistantDeltaEvent); ok && len(replaced) > 0 {
			afterReplaced++
		}
	}
	require.Len(t, replaced, 1, "the corrected reply is sent once")
	assert.Equal(t, reply, replaced[0].Text, "the reader is left with what was recorded")
	assert.Equal(t, reason, replaced[0].Reason)
	assert.Zero(t, afterReplaced, "no streamed piece follows the replacement")
}

// A reply the passes leave alone is not withdrawn or sent again.
func TestDrive_LeavesACleanReplyAlone(t *testing.T) {
	t.Parallel()

	kept := runtimetest.KeptShipments(25)
	reply := "Eight loads are late; " + replycheck.ArtifactRef(&kept) + " lists them."

	result, events := passRun(t, reply, &kept)

	assert.Equal(t, reply, result.Reply)
	assert.Empty(t, regroundedActions(events))
	for _, event := range *events {
		assert.NotEqual(t, assistantports.AssistantEventReplyReplaced, event.Event)
		assert.NotEqual(t, assistantports.AssistantEventRetrying, event.Event)
	}
}

func TestFoldReplaced_PointsAnUnreadResultAtTheTableItsCardWentInto(t *testing.T) {
	t.Parallel()

	read := &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "entity_card",
		Title: "Driver settlement",
	}
	unread := &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "entity_card",
		Title: "Driver settlement",
	}
	table := &agentruntimeports.ShownArtifact{
		ID:       pulid.MustNew("art_"),
		Kind:     "table_view",
		Title:    "Driver settlements (3)",
		Rows:     3,
		Replaces: []pulid.ID{read.ID, unread.ID},
	}
	readContent := "Jane Doe's settlement" + runtimetools.ShownNote(read)
	unreadContent := "Sarah Williams's settlement" + runtimetools.ShownNote(unread)

	turn := &Turn{
		messages: []llmports.Message{
			{Role: llmports.RoleTool, ToolCallID: "call_read", Content: readContent},
			{Role: llmports.RoleAssistant},
			{Role: llmports.RoleTool, ToolCallID: "call_unread", Content: unreadContent},
		},
		artifacts: []agentruntimeports.ShownArtifact{*read, *unread, *table},
		result: &agentruntimeports.RunResult{Messages: []conversation.Message{
			{Role: conversation.RoleTool, ToolCallID: "call_unread", Content: unreadContent},
		}},
	}

	turn.foldReplaced(table)

	assert.Equal(t, readContent, turn.messages[0].Content,
		"a result the model has read is never rewritten")
	folded := "Sarah Williams's settlement" + runtimetools.FoldedNote(table)
	assert.Equal(t, folded, turn.messages[2].Content)
	assert.Equal(t, folded, turn.result.Messages[0].Content)
	assert.NotContains(t, turn.messages[2].Content, unread.ID.String())
	require.Len(t, turn.artifacts, 1)
	assert.Equal(t, table.ID, turn.artifacts[0].ID)
}
