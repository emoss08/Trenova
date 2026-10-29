package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type boardObserver struct {
	boards []agentruntimeports.BoardRequest
	shown  *agentruntimeports.ShownArtifact
	err    error
}

func (o *boardObserver) observe(
	observation agentruntimeports.ToolObservation,
) (*agentruntimeports.ShownArtifact, error) {
	request, ok := observation.Data.(agentruntimeports.BoardRequest)
	if !ok {
		return nil, nil
	}
	o.boards = append(o.boards, request)

	return o.shown, o.err
}

func listedShipments(ids ...string) agenttoolports.AgentQueryTool {
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, map[string]any{"id": id, "proNumber": "S2610-" + id[len(id)-4:]})
	}

	return queryTool("shipment_list", map[string]any{"items": items}, nil)
}

func TestRun_OffersABoardWhereArtifactsAreKept(t *testing.T) {
	t.Parallel()

	observer := &boardObserver{}
	_, completion := runWithObserver(t, observer.observe, nil, runtimetest.TextTurn("Here it is."))

	assert.Contains(t, offeredNames(completion.LastReq.Tools), runtimetools.ShowBoardName)
}

func TestRun_ShowsABoardOfRecordsTheConversationRead(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	board := pulid.MustNew("art_")
	observer := &boardObserver{shown: &agentruntimeports.ShownArtifact{
		ID: board, Kind: "board", Title: "Late shipments on the Dallas lanes",
		Brief: "# Late shipments on the Dallas lanes\n\n| Shipment | Detail | Status | Behind |",
	}}
	result, _ := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(first, second)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn(runtimetools.ShowBoardName, runtimetest.ShipmentBoard(first, second)),
		runtimetest.TextTurn("The board is beside the conversation."),
	)

	require.Len(t, observer.boards, 1)
	shown := observer.boards[0].Board
	assert.Equal(t, "Late shipments on the Dallas lanes", shown.Title)
	require.Len(t, shown.Blocks, 2)
	assert.Len(t, shown.Blocks[0].Records.IDs, 2)

	content := result.Messages[4].Content
	assert.False(t, result.Messages[4].ToolFailed)
	assert.Contains(t, content, board.String())
	assert.Contains(t, content, "| Shipment | Detail | Status | Behind |")
	assert.Contains(t, content, "do not repeat its figures or rows")
}

func TestRun_RefusesABoardNamingARecordNoToolReturned(t *testing.T) {
	t.Parallel()

	read, invented := pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()
	observer := &boardObserver{}
	result, _ := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(read)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn(runtimetools.ShowBoardName, runtimetest.ShipmentBoard(read, invented)),
		runtimetest.TextTurn("Sorry."),
	)

	assert.Empty(t, observer.boards)
	assert.True(t, result.Messages[4].ToolFailed)
	assert.Contains(t, result.Messages[4].Content, invented)
	assert.Contains(t, result.Messages[4].Content, "no tool in this conversation returned")
	assert.NotContains(t, result.Messages[4].Content, read)
}

func TestRun_RefusesABoardWhoseWordsStateFigures(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	args := runtimetest.ShipmentBoard(id)
	args["verdict"] = "The two cost $48,210.55 in [lost revenue][1]."
	observer := &boardObserver{}
	result, _ := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn(runtimetools.ShowBoardName, args),
		runtimetest.TextTurn("Sorry."),
	)

	assert.Empty(t, observer.boards)
	assert.True(t, result.Messages[4].ToolFailed)
	assert.Contains(t, result.Messages[4].Content,
		`verdict states "$48,210.55 in"; the board reads every figure live from its blocks`)
}

func TestRun_RefusesABoardThatDoesNotParse(t *testing.T) {
	t.Parallel()

	observer := &boardObserver{}
	result, _ := runWithObserver(
		t,
		observer.observe,
		nil,
		runtimetest.ToolTurn(
			runtimetools.ShowBoardName,
			map[string]any{"title": "Board", "verdict": "x"},
		),
		runtimetest.TextTurn("Sorry."),
	)

	assert.Empty(t, observer.boards)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "blocks needs at least one block")
}

func TestRun_AFixableBoardIsSentAgain(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_").String()
	observer := &boardObserver{err: &agentruntimeports.PublishMistake{
		Reason: "the person may not read shipments, so leave the blocks that name them out",
	}}
	result, _ := runWithObserver(t, observer.observe,
		[]agenttoolports.AgentQueryTool{listedShipments(id)},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": "late"}),
		runtimetest.ToolTurn(runtimetools.ShowBoardName, runtimetest.ShipmentBoard(id)),
		runtimetest.TextTurn("Sorry."),
	)

	content := result.Messages[4].Content
	assert.True(t, result.Messages[4].ToolFailed)
	assert.Contains(t, content, "the person may not read shipments")
	assert.Contains(t, content, "Fix the call and send it again")
}

func TestShownArtifact_ABoardMustBeLinked(t *testing.T) {
	t.Parallel()

	assert.True(t, (&agentruntimeports.ShownArtifact{Kind: "board"}).MustBeLinked())
}
