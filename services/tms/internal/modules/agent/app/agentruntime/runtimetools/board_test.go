package runtimetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoardStepOutcome_ReadsTheBoardItRevises(t *testing.T) {
	t.Parallel()

	revises := pulid.MustNew("art_")
	args := runtimetest.ShipmentBoard(pulid.MustNew("shp_").String())
	args["boardId"] = revises.String()

	outcome := BoardStepOutcome(args)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, revises, request.ArtifactID)
	assert.True(t, outcome.Publishes)
}

func TestBoardStepOutcome_KeepsThePlaceItWasAskedFor(t *testing.T) {
	t.Parallel()

	args := runtimetest.ShipmentBoard(pulid.MustNew("shp_").String())
	args["place"] = "beside"

	outcome := BoardStepOutcome(args)

	require.False(t, outcome.Failed, outcome.Content)
	request, ok := outcome.Data.(agentruntimeports.BoardRequest)
	require.True(t, ok)
	assert.Equal(t, board.PlaceBeside, request.Asked)
	assert.Equal(t, board.PlaceBeside, request.Board.Place)
}

func TestConversationBoard_IsABoardShownInTheConversation(t *testing.T) {
	t.Parallel()

	inline := &agentruntimeports.ShownArtifact{Kind: "board", Place: "conversation"}
	assert.Same(t, inline, ConversationBoard(&toolcall.Result{Shown: inline}))
	assert.Nil(t, ConversationBoard(&toolcall.Result{Shown: inline, Failed: true}))
	assert.Nil(t, ConversationBoard(&toolcall.Result{
		Shown: &agentruntimeports.ShownArtifact{Kind: "board", Place: "beside"},
	}))
	assert.Nil(t, ConversationBoard(&toolcall.Result{
		Shown: &agentruntimeports.ShownArtifact{Kind: "document", Place: "conversation"},
	}))
	assert.Nil(t, ConversationBoard(&toolcall.Result{}))
}

func TestBoardReply_IsTheAnswerAndItsLink(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("art_")
	shown := &agentruntimeports.ShownArtifact{
		ID: id, Kind: "board", Title: "Six late [shipments]", Answer: "Six are late.",
	}
	assert.Equal(t, "Six are late ([Six late shipments](artifact:"+id.String()+")).",
		BoardReply(shown))
	assert.Empty(t, replycheck.StrayArtifactLinks(BoardReply(shown)))

	shown.Answer = "  "
	assert.Equal(t, "[Six late shipments](artifact:"+id.String()+")", BoardReply(shown))
}

func TestBoardContent_SaysWhereTheBoardIs(t *testing.T) {
	t.Parallel()

	shown := &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "board", Title: "Late loads", Place: "conversation",
	}
	inline := BoardContent("show_board", shown, "")
	assert.Contains(t, inline, "Result from show_board:")
	assert.Contains(t, inline, `"status":"shown","place":"conversation"`)
	assert.Contains(t, BoardContent(ShowAnswerTool, shown, ""), "Result from show_answer:")
	assert.Contains(t, inline, "the turn ended with it")

	shown.Place = "beside"
	moved := BoardContent("show_board", shown, board.PlaceConversation)
	assert.Contains(t, moved, `"place":"beside"`)
	assert.Contains(t, moved, "You asked for it in the conversation")
	assert.NotContains(t, BoardContent("show_board", shown, ""), "You asked for it")

	shown.Place = ""
	assert.Contains(t, BoardContent("show_board", shown, ""), `"place":"beside"`)
}
