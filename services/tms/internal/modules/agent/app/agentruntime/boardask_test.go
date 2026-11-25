package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type boardAskRun struct {
	result     *agentruntimeports.RunResult
	events     *[]kernelports.StreamEvent
	completion *scriptedCompletion
}

func runBoardAsk(
	t *testing.T,
	observer agentruntimeports.ToolObserver,
	actions []agenttoolports.AgentTool,
	tools []agenttoolports.AgentQueryTool,
	turns ...*llmports.ChatCompletionResult,
) boardAskRun {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(
		completion,
		&stubQueryRegistry{Tools: tools},
		&stubActionRegistry{Tools: actions},
		nil,
	)
	names := make([]string, 0, len(tools)+len(actions))
	for _, tool := range tools {
		names = append(names, tool.Name())
	}
	for _, tool := range actions {
		names = append(names, tool.Name())
	}
	req := &agentruntimeports.RunRequest{
		Definition:   runtimetest.Definition(names...),
		Actor:        runtimetest.Actor(),
		Input:        "Which of the late loads is further behind?",
		ToolObserver: observer,
	}
	events := recordEvents(req)

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return boardAskRun{result: result, events: events, completion: completion}
}

func shipmentLabel(id string) string { return "S2610-" + id[len(id)-4:] }

func TestRun_AsksOnceForABoardThenEndsTheTurnOnIt(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	shown := shownBoard("conversation")
	observer := &boardObserver{shown: shown}
	prose := shipmentLabel(first) + " is the furthest behind, and " + shipmentLabel(second) +
		" is close behind it."

	run := runBoardAsk(t, observer.observe, nil,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list_view", map[string]any{"query": "late"}),
		runtimetest.TextTurn(prose),
		runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(first, second)),
		runtimetest.TextTurn("This is never asked for."),
	)

	assert.Equal(t, 3, run.completion.CallCount, "asked once for a board, not after it")
	assert.Contains(t, run.result.Reply, "artifact:"+shown.ID.String())
	assert.Equal(t, 1, restarts(*run.events))
	assert.Equal(t,
		[]assistantstreamports.RegroundAction{assistantstreamports.RegroundAnswerWithBoard},
		regroundedActions(run.events),
	)
	asked := run.completion.Requests[2]
	assert.Equal(t, ReaskBoard, asked.Reask)
	draft, correction := asked.Messages[len(asked.Messages)-2], asked.Messages[len(asked.Messages)-1]
	assert.Equal(t, llmports.RoleAssistant, draft.Role)
	assert.Equal(t, prose, draft.Content)
	assert.Equal(t, llmports.RoleSystem, correction.Role)
	assert.Contains(t, correction.Content,
		"it names "+shipmentLabel(first)+", "+shipmentLabel(second)+".")
	assert.NotContains(t, correction.Content, first)
	assert.Contains(t, run.result.Messages[2].Content, "\n\n[These are 2 records.",
		"the list's result carries the note in the transcript")
	assert.Contains(t, run.completion.Requests[1].Messages[len(
		run.completion.Requests[1].Messages,
	)-1].Content, "[These are 2 records.", "and where the model reads it")
}

func TestRun_TakesASecondProseReplyAsItIs(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	observer := &boardObserver{shown: shownBoard("conversation")}
	prose := shipmentLabel(first) + " is the furthest behind, and " + shipmentLabel(second) +
		" is close behind it."
	again := shipmentLabel(first) + " is further behind than " + shipmentLabel(second) + "."

	run := runBoardAsk(t, observer.observe, nil,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list_view", map[string]any{"query": "late"}),
		runtimetest.TextTurn(prose),
		runtimetest.TextTurn(again),
	)

	assert.Equal(t, 3, run.completion.CallCount)
	assert.Equal(t, again, run.result.Reply)
	assert.Equal(t, 1, restarts(*run.events), "the person sees one restart")
	assert.Empty(t, observer.boards)
}

func tableObserver(
	table, board *agentruntimeports.ShownArtifact,
) agentruntimeports.ToolObserver {
	return func(observation agentruntimeports.ToolObservation) (
		*agentruntimeports.ShownArtifact, error,
	) {
		if _, boards := observation.Data.(agentruntimeports.BoardRequest); boards {
			return board, nil
		}
		if observation.Call.Name == "shipment_list_view" {
			return table, nil
		}

		return nil, nil
	}
}

func TestRun_LeavesAReplyThatLinksTheTable(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	table := runtimetest.KeptShipments(2)
	reply := shipmentLabel(first) + " is further behind than " + shipmentLabel(second) +
		", as " + replycheck.ArtifactRef(&table) + " shows."

	run := runBoardAsk(t, tableObserver(&table, nil), nil,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list_view", map[string]any{"query": "late"}),
		runtimetest.TextTurn(reply),
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, reply, run.result.Reply)
	assert.Zero(t, restarts(*run.events))
	result := run.result.Messages[2].Content
	assert.True(t, strings.HasSuffix(result, "one record or one fact is a sentence.]"),
		"the board note is the last thing on the result, after the table's note")
	assert.Less(t, strings.Index(result, "kept"), strings.Index(result, "[These are 2 records."))
}

func TestRun_LeavesAReplyThatGivesACountAndLinksTheList(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	table := runtimetest.KeptShipments(2)
	reply := "Two loads are late; they are in " + replycheck.ArtifactRef(&table) + "."

	run := runBoardAsk(t, tableObserver(&table, nil), nil,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list_view", map[string]any{"query": "late"}),
		runtimetest.TextTurn(reply),
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Equal(t, reply, run.result.Reply)
	assert.Zero(t, restarts(*run.events))
}

func TestRun_LeavesAReplyAboutAProposedChange(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	comment := &agenttest.StubActionTool{
		ToolName: "shipment_add_comment",
		Tier:     agent.TierPropose,
		Schema:   runtimetest.ClosedSchema("shipmentId", "comment"),
	}
	reply := "Notes on " + shipmentLabel(first) + " and " + shipmentLabel(second) +
		" wait for your approval."

	run := runBoardAsk(t, (&boardObserver{}).observe,
		[]agenttoolports.AgentTool{comment},
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list_view", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("shipment_add_comment", map[string]any{
			"shipmentId": first, "comment": "Call before delivery",
		}),
		runtimetest.TextTurn(reply),
	)

	require.NotEmpty(t, run.result.Actions)
	assert.Equal(t, 3, run.completion.CallCount)
	assert.Equal(t, reply, run.result.Reply)
	assert.Zero(t, restarts(*run.events))
}
