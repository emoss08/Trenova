package benchcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boardTurn(
	t *testing.T,
	place board.Place,
	readings int,
	reply string,
) (*benchrecord.TurnRecord, pulid.ID) {
	t.Helper()

	layout := &board.Board{Title: "Two loads side by side", Verdict: "Words.", Place: place}
	for range readings {
		layout.Readings = append(layout.Readings, board.Reading{Key: "late"})
	}
	payload, err := board.NewPayload(layout, board.Snapshot{}, "Dispatch desk").Map()
	require.NoError(t, err)
	id := pulid.MustNew("art_")

	return &benchrecord.TurnRecord{
		Reply: reply,
		Artifacts: []assistantartifactports.AssistantArtifact{{
			ID: id, Kind: assistantartifact.KindBoard, Title: layout.Title, Payload: payload,
		}},
	}, id
}

func boardReply(id pulid.ID) string {
	return runtimetools.BoardReply(&agentruntimeports.ShownArtifact{
		ID:     id,
		Kind:   string(assistantartifact.KindBoard),
		Title:  "Two loads side by side",
		Place:  string(board.PlaceConversation),
		Answer: "SEED-SHP-010 is further behind, by 2h.",
	})
}

func TestArtifactLinkChecks_PassesTheReplyAConversationBoardEndsOn(t *testing.T) {
	t.Parallel()

	turn, id := boardTurn(t, board.PlaceConversation, 1, "")
	turn.Reply = boardReply(id)

	assert.Empty(t, replycheck.StrayArtifactLinks(turn.Reply), turn.Reply)
	checks := artifactLinkChecks([]*benchrecord.TurnRecord{turn})
	require.NotEmpty(t, checks)
	assert.True(t, checks[0].Passed, checks[0].Detail)
	assert.True(t, checks[1].Passed, checks[1].Detail)
}

func TestArtifactLinkChecks_CountsALinkStandingAfterTheSentence(t *testing.T) {
	t.Parallel()

	turn, id := boardTurn(t, board.PlaceConversation, 1, "")
	turn.Reply = "SEED-SHP-010 is further behind, by 2h. ([Two loads side by side](artifact:" +
		id.String() + "))"

	checks := artifactLinkChecks([]*benchrecord.TurnRecord{turn})

	require.NotEmpty(t, checks)
	assert.False(t, checks[0].Passed)
}

func TestBoardCheck_PassesAConversationBoardTheReplyEndsOn(t *testing.T) {
	t.Parallel()

	turn, id := boardTurn(t, board.PlaceConversation, 2, "")
	turn.Reply = boardReply(id)
	want := &benchscenario.ExpectedBoard{Place: "conversation", MinReadings: 1}

	check := boardCheck(want, []*benchrecord.TurnRecord{turn})

	assert.True(t, check.Passed, check.Detail)
}

func TestBoardCheck_FailsWhatIsNotTheBoardAskedFor(t *testing.T) {
	t.Parallel()

	want := &benchscenario.ExpectedBoard{Place: "conversation", MinReadings: 1}

	beside, id := boardTurn(t, board.PlaceBeside, 1, "")
	beside.Reply = boardReply(id)
	check := boardCheck(want, []*benchrecord.TurnRecord{beside})
	assert.False(t, check.Passed)
	assert.Contains(t, check.Detail, "beside with 1 readings")

	bare, id := boardTurn(t, board.PlaceConversation, 0, "")
	bare.Reply = boardReply(id)
	assert.False(t, boardCheck(want, []*benchrecord.TurnRecord{bare}).Passed)

	unlinked, _ := boardTurn(t, board.PlaceConversation, 1, "SEED-SHP-010 is further behind.")
	assert.False(t, boardCheck(want, []*benchrecord.TurnRecord{unlinked}).Passed)

	none := boardCheck(want, []*benchrecord.TurnRecord{{Reply: "Nothing."}})
	assert.False(t, none.Passed)
	assert.Equal(t, "no board was shown", none.Detail)
}

func TestNoBoardCheck_PassesAStepThatShowedNoBoardAndNamesOneThatDid(t *testing.T) {
	t.Parallel()

	prose := &benchrecord.TurnRecord{Reply: "SEED-SHP-009 is ready to invoice."}
	assert.True(t, noBoardCheck([]*benchrecord.TurnRecord{prose}).Passed)

	beside, _ := boardTurn(t, board.PlaceBeside, 0, "It is beside the conversation.")
	check := noBoardCheck([]*benchrecord.TurnRecord{prose, beside})
	assert.False(t, check.Passed, "a board beside the conversation is a board too")
	assert.Equal(t, `"Two loads side by side"`, check.Detail)
	assert.Equal(t, "no board was shown", check.Name)
}

func TestBoardCheck_ReadsEachMeasureTheQuestionAskedFor(t *testing.T) {
	t.Parallel()

	read := func(key string, measure board.Measure, aggregate board.Aggregate) board.Reading {
		return board.Reading{Key: key, FigureSource: board.FigureSource{
			Measure: measure, Aggregate: aggregate,
		}}
	}
	want := &benchscenario.ExpectedBoard{
		Place: string(board.PlaceConversation), MinReadings: 1,
		Reads: []string{"lateness", "revenue"},
	}
	tests := []struct {
		name     string
		readings []board.Reading
		passed   bool
	}{
		{
			name: "lateness and revenue each read",
			readings: []board.Reading{
				read("largest", board.MeasureLateness, board.AggregateMax),
				read("total_1_revenue", board.MeasureRevenue, board.AggregateSum),
			},
			passed: true,
		},
		{
			name: "only lateness, even where the words speak of money",
			readings: []board.Reading{
				read("largest", board.MeasureLateness, board.AggregateMax),
				read("total", board.MeasureLateness, board.AggregateSum),
			},
		},
		{
			name: "revenue only counted, never read",
			readings: []board.Reading{
				read("largest", board.MeasureLateness, board.AggregateMax),
				read("count", board.MeasureRevenue, board.AggregateCount),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			turn, id := boardTurn(t, board.PlaceConversation, 0, "")
			layout := &board.Board{
				Title: "Two loads side by side", Verdict: "Words.",
				Place: board.PlaceConversation, Readings: tt.readings,
			}
			payload, err := board.NewPayload(layout, board.Snapshot{}, "Dispatch desk").Map()
			require.NoError(t, err)
			turn.Artifacts[0].Payload = payload
			turn.Reply = boardReply(id)

			check := boardCheck(want, []*benchrecord.TurnRecord{turn})

			assert.Equal(t, tt.passed, check.Passed, check.Detail)
		})
	}
}
