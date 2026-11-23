package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shownBoard(place string) *agentruntimeports.ShownArtifact {
	return &agentruntimeports.ShownArtifact{
		ID:     pulid.MustNew("art_"),
		Kind:   "board",
		Title:  "Late shipments on the Dallas lanes",
		Place:  place,
		Answer: "Two of the 2 shipments are held at the dock.",
		Brief:  "# Late shipments on the Dallas lanes",
	}
}

func lastReply(t *testing.T, result *agentruntimeports.RunResult) string {
	t.Helper()
	require.NotEmpty(t, result.Messages)

	return result.Messages[len(result.Messages)-1].Content
}

func TestRun_EndsTheTurnOnceABoardIsInTheConversation(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	shown := shownBoard("conversation")
	observer := &boardObserver{shown: shown}
	showing := runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(id))
	showing.InputTokens = 1200
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		showing,
		runtimetest.TextTurn("This is never asked for."),
	)

	assert.Equal(t, 2, completion.CallCount, "the model is not asked again after the board")
	assert.Equal(t, 1200, result.Messages[3].InputTokens)
	want := "Two of the 2 shipments are held at the dock ([Late shipments on the Dallas " +
		"lanes](artifact:" + shown.ID.String() + "))."
	assert.Equal(t, want, result.Reply)
	assert.Equal(t, want, lastReply(t, result))
	last := result.Messages[len(result.Messages)-1]
	assert.Zero(t, last.InputTokens, "the step's own message already holds its usage")
	assert.Nil(t, last.CostUSD)
	assert.Equal(t, 2, result.ToolCallsUsed)

	content := result.Messages[4].Content
	assert.Contains(t, content, `"place":"conversation"`)
	assert.Contains(t, content, "drawn in the conversation as your reply")
}

func TestRun_KeepsWorkingAfterABoardBesideTheConversation(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	shown := shownBoard("beside")
	observer := &boardObserver{shown: shown}
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(id)),
		runtimetest.TextTurn("The board is beside the conversation."),
	)

	assert.Equal(t, 3, completion.CallCount, "the model is asked for its reply after the board")
	assert.Equal(t, "The board is beside the conversation ([Late shipments on the Dallas "+
		"lanes](artifact:"+shown.ID.String()+")).", result.Reply,
		"the model's own sentence, with the link it left out joined by placement")
	assert.Contains(t, result.Messages[4].Content, `"place":"beside"`)
	assert.Contains(t, result.Messages[4].Content, "do not repeat its figures or rows")
}

func TestRun_SaysWhyABoardAskedForInTheConversationWentBeside(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	args := runtimetest.ShipmentBoard(id)
	args["place"] = "conversation"
	for _, text := range []string{"One waits on a driver.", "One on the dock.", "Call both."} {
		args["blocks"] = append(args["blocks"].([]any), map[string]any{
			"type": "prose", "text": text,
		})
	}
	observer := &boardObserver{shown: shownBoard("beside")}
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_board", args),
		runtimetest.TextTurn("It is beside the conversation."),
	)

	assert.Equal(t, 3, completion.CallCount)
	require.Len(t, observer.boards, 1)
	assert.Equal(t, "conversation", string(observer.boards[0].Asked))
	assert.Equal(t, "beside", string(observer.boards[0].Board.Place))
	assert.Contains(t, result.Messages[4].Content,
		"You asked for it in the conversation, but it holds more than 4 blocks")
}

func TestRun_AFailedBoardEndsNothing(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	observer := &boardObserver{err: &agentruntimeports.PublishMistake{
		Reason: "the person may not read shipments",
	}}
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(id)),
		runtimetest.TextTurn("I could not show the board."),
	)

	assert.Equal(t, 3, completion.CallCount)
	assert.True(t, result.Messages[4].ToolFailed)
	assert.Equal(t, "I could not show the board.", result.Reply)
}

func TestRun_ARefusedBoardEndsNothing(t *testing.T) {
	t.Parallel()

	read, invented := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	observer := &boardObserver{shown: shownBoard("conversation")}
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(read)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(read, invented)),
		runtimetest.TextTurn("Sorry."),
	)

	assert.Empty(t, observer.boards)
	assert.Equal(t, 3, completion.CallCount)
	assert.Equal(t, "Sorry.", result.Reply)
}

func TestRun_ABoardAndAQuestionInOneStepEndOnTheBoard(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	shown := shownBoard("conversation")
	observer := &boardObserver{shown: shown}
	both := runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(id))
	both.ToolCalls = append(both.ToolCalls, llmports.ToolCall{
		ID:        "call_2",
		Name:      "ask_user",
		Arguments: map[string]any{"question": "Re-tender the furthest one?", "allowOther": true},
	})
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		both,
		runtimetest.TextTurn("This is never asked for."),
	)

	assert.Equal(t, 2, completion.CallCount)
	results := 0
	for idx := range result.Messages {
		if result.Messages[idx].ToolCallID != "" {
			results++
		}
	}
	assert.Equal(t, 3, results, "every call of the step is answered before the turn ends")
	assert.Contains(t, result.Reply, "artifact:"+shown.ID.String())
}

func TestRun_EndsTheTurnOnAShownAnswer(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	shown := shownBoard("conversation")
	observer := &boardObserver{shown: shown}
	result, completion := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_answer", map[string]any{
			"title":   "Late shipments on the Dallas lanes",
			"answer":  "There are {count} late loads, the furthest {largest} behind.",
			"measure": "lateness",
			"groups": []any{map[string]any{
				"title": "Held at the dock", "records": []any{first, second},
			}},
		}),
		runtimetest.TextTurn("This is never asked for."),
	)

	assert.Equal(t, 2, completion.CallCount, "the model is not asked again after the answer")
	require.Len(t, observer.boards, 1)
	request := observer.boards[0]
	assert.Equal(t, "conversation", string(request.Board.Place))
	assert.Equal(t, "conversation", string(request.Asked))
	assert.Len(t, request.Board.Readings, 2)
	assert.Contains(t, result.Reply, "artifact:"+shown.ID.String())
	assert.Contains(t, result.Messages[4].Content, "Result from show_answer:")
}

func TestRun_PutsABoardWithNoPlaceBeside(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	observer := &boardObserver{shown: shownBoard("beside")}
	runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn("show_board", runtimetest.ShipmentBoard(id)),
		runtimetest.TextTurn("It is beside the conversation."),
	)

	require.Len(t, observer.boards, 1)
	assert.Equal(t, "beside", string(observer.boards[0].Board.Place))
}

func TestRun_OffersShowAnswerBesideShowBoard(t *testing.T) {
	t.Parallel()

	_, completion := runWithObserver(t, (&boardObserver{}).observe, nil,
		runtimetest.TextTurn("Here it is."))

	offered := offeredNames(completion.LastReq.Tools)
	assert.Contains(t, offered, "show_answer")
	assert.Contains(t, offered, "show_board")
}
