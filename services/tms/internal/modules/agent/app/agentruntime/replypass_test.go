package agentruntime

import (
	"fmt"
	"strings"
	"testing"

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

func keptShipments(rows int) agentruntimeports.ShownArtifact {
	labels := make([]string, 0, rows)
	for idx := range rows {
		labels = append(labels, fmt.Sprintf("PRO-%d", idx+1))
	}

	return agentruntimeports.ShownArtifact{
		ID:     pulid.MustNew("art_"),
		Kind:   "table_view",
		Title:  "Late shipments",
		Rows:   rows,
		Labels: labels,
	}
}

func markdownReprint(rows int) string {
	var b strings.Builder
	b.WriteString("Here they are:\n| Pro | Status |\n| --- | --- |\n")
	for idx := range rows {
		fmt.Fprintf(&b, "| **PRO-%d** | Late |\n", idx+1)
	}
	b.WriteString("\nCall the carriers first.")

	return b.String()
}

// A reprint of a kept table is replaced by one sentence pointing to it, and
// the prose around it stays.
func TestPointToTables_ReplacesAReprintWithAPointer(t *testing.T) {
	t.Parallel()

	kept := keptShipments(25)

	got, table := pointToTables(markdownReprint(8), []agentruntimeports.ShownArtifact{kept})

	require.NotNil(t, table)
	assert.Equal(t, kept.ID, table.ID)
	assert.Equal(t, "Here they are: "+ArtifactRef(&kept)+"\n\nCall the carriers first.", got)
	assert.Empty(t, StrayArtifactLinks(got))
}

// A few rows are an answer, a table of something else is not a reprint, and
// a table the reply already points to has been pointed to.
func TestPointToTables_LeavesWhatIsNotAReprint(t *testing.T) {
	t.Parallel()

	kept := keptShipments(25)

	short := markdownReprint(minReprintRows - 1)
	got, table := pointToTables(short, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, short, got)
	assert.Nil(t, table)

	other := "By customer:\n| Customer | Shipments |\n| --- | --- |\n" +
		strings.Repeat("| Acme | 3 |\n", 7)
	got, table = pointToTables(other, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, other, got, "a per-customer summary is not the shipments table")
	assert.Nil(t, table)

	linked := "See " + ArtifactRef(&kept) + ".\n" + markdownReprint(8)
	got, table = pointToTables(linked, []agentruntimeports.ShownArtifact{kept})
	assert.Equal(t, linked, got)
	assert.Nil(t, table)
}

// The title is enough when the rows' values were not kept: the line above
// the reprint and its first header name what the table is.
func TestPointToTables_RecognisesAReprintByItsTitle(t *testing.T) {
	t.Parallel()

	kept := keptShipments(0)
	reply := "The late shipments:\n| Shipment | Status |\n| --- | --- |\n" +
		strings.Repeat("| X | Late |\n", 7)

	_, table := pointToTables(reply, []agentruntimeports.ShownArtifact{kept})

	require.NotNil(t, table)
	assert.Equal(t, kept.ID, table.ID)
}

func passRun(t *testing.T, reply string, shown *agentruntimeports.ShownArtifact) (
	*agentruntimeports.RunResult,
	*[]kernelports.StreamEvent,
) {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("shipment_list", map[string]any{}),
		textTurn(reply),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		queryTool("shipment_list", map[string]any{"items": []any{}}, nil),
	}}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: testDefinition("shipment_list"),
		Actor:      testActor(),
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

	kept := keptShipments(25)
	reply := "PRO-1 (ID **shp_01M3Q2Y4SRFE0YW60JY6F5NRW7**) is the oldest.\n" + markdownReprint(8)

	result, events := passRun(t, reply, &kept)

	assert.NotContains(t, result.Reply, "shp_01M3Q2Y4SRFE0YW60JY6F5NRW7")
	assert.NotContains(t, result.Reply, "| --- |")
	assert.Contains(t, result.Reply, ArtifactRef(&kept))
	assert.Equal(t, result.Reply, result.Messages[len(result.Messages)-1].Content)
	assert.Equal(t, []assistantports.RegroundAction{
		assistantports.RegroundStripIDs,
		assistantports.RegroundPointToTable,
	}, regroundedActions(events))

	assertReplacedWithoutRestart(t, *events, result.Reply, replyPassReplacedReason)
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

	kept := keptShipments(25)
	reply := "Eight loads are late; " + ArtifactRef(&kept) + " lists them."

	result, events := passRun(t, reply, &kept)

	assert.Equal(t, reply, result.Reply)
	assert.Empty(t, regroundedActions(events))
	for _, event := range *events {
		assert.NotEqual(t, assistantports.AssistantEventReplyReplaced, event.Event)
		assert.NotEqual(t, assistantports.AssistantEventRetrying, event.Event)
	}
}

// A reply that reprinted two kept tables keeps neither copy: each is pointed
// to once, in its own place, and the prose between them stays.
func TestPointToTables_PointsEveryReprintToItsOwnTable(t *testing.T) {
	t.Parallel()

	late := keptShipments(25)
	carriers := agentruntimeports.ShownArtifact{
		ID:     pulid.MustNew("art_"),
		Kind:   "table_view",
		Title:  "Carriers to call",
		Rows:   10,
		Labels: []string{"Acme", "Bolt", "Cargo", "Delta", "Eagle", "Falcon", "Giant", "Hawk"},
	}
	var carrierRows strings.Builder
	carrierRows.WriteString("| Carrier | Loads |\n| --- | --- |\n")
	for _, name := range carriers.Labels {
		fmt.Fprintf(&carrierRows, "| %s | 2 |\n", name)
	}
	reply := markdownReprint(8) + "\n\n" + carrierRows.String() + "\nThat is all."

	got, first := pointToTables(reply, []agentruntimeports.ShownArtifact{late, carriers})

	require.NotNil(t, first)
	assert.Equal(t, late.ID, first.ID, "the first table pointed to is reported")
	assert.Contains(t, got, "Here they are: "+ArtifactRef(&late))
	assert.Contains(t, got, "Call the carriers first ("+ArtifactRef(&carriers)+").")
	assert.Empty(t, StrayArtifactLinks(got))
	assert.NotContains(t, got, "| PRO-1")
	assert.NotContains(t, got, "| Acme |")
	assert.Contains(t, got, "Call the carriers first.")
	assert.Contains(t, got, "That is all.")
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
	readContent := "Jane Doe's settlement" + shownNote(read)
	unreadContent := "Sarah Williams's settlement" + shownNote(unread)

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
	folded := "Sarah Williams's settlement" + foldedNote(table)
	assert.Equal(t, folded, turn.messages[2].Content)
	assert.Equal(t, folded, turn.result.Messages[0].Content)
	assert.NotContains(t, turn.messages[2].Content, unread.ID.String())
	require.Len(t, turn.artifacts, 1)
	assert.Equal(t, table.ID, turn.artifacts[0].ID)
}
