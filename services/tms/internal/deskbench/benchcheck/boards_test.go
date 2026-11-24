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
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
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
	payload, err := (&board.Payload{Board: *layout, Place: layout.Placed(), Author: "Dispatch desk"}).Map()
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
				read("largest", board.Measure("lateness"), board.AggregateMax),
				read("total_1_revenue", board.Measure("revenue"), board.AggregateSum),
			},
			passed: true,
		},
		{
			name: "only lateness, even where the words speak of money",
			readings: []board.Reading{
				read("largest", board.Measure("lateness"), board.AggregateMax),
				read("total", board.Measure("lateness"), board.AggregateSum),
			},
		},
		{
			name: "revenue only counted, never read",
			readings: []board.Reading{
				read("largest", board.Measure("lateness"), board.AggregateMax),
				read("count", board.Measure("revenue"), board.AggregateCount),
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
			payload, err := (&board.Payload{Board: *layout, Place: layout.Placed(), Author: "Dispatch desk"}).Map()
			require.NoError(t, err)
			turn.Artifacts[0].Payload = payload
			turn.Reply = boardReply(id)

			check := boardCheck(want, []*benchrecord.TurnRecord{turn})

			assert.Equal(t, tt.passed, check.Passed, check.Detail)
		})
	}
}

func TestBoardCheck_ReadsTheKindsMetricsAndFiguresTheQuestionAskedFor(t *testing.T) {
	t.Parallel()

	credential := recordkind.Kind(permission.ResourceWorkerCredential)
	want := &benchscenario.ExpectedBoard{
		Place:    string(board.PlaceConversation),
		Kinds:    []string{string(credential)},
		Measures: []string{"expiry"},
		Metrics:  []string{billingboard.MetricReceivables},
		Figures:  []string{"arOverdue:" + billingboard.MetricReceivables},
	}
	shown := func(layout *board.Board) []*benchrecord.TurnRecord {
		turn, id := boardTurn(t, board.PlaceConversation, 0, "")
		layout.Title, layout.Verdict, layout.Place = "Credentials", "Words.", board.PlaceConversation
		payload, err := (&board.Payload{
			Board: *layout, Place: board.PlaceConversation, Author: "Credential desk",
		}).Map()
		require.NoError(t, err)
		turn.Artifacts[0].Payload = payload
		turn.Reply = boardReply(id)

		return []*benchrecord.TurnRecord{turn}
	}
	records := board.Block{
		Type:    board.BlockRecords,
		Records: &board.RecordSet{Kind: credential, IDs: []pulid.ID{pulid.MustNew("wcred_")}},
		Measure: "expiry",
	}
	metric := board.Block{Type: board.BlockMetric, Metric: billingboard.MetricReceivables}
	figure := board.Reading{Key: "overdue", FigureSource: board.FigureSource{
		Metric: billingboard.MetricReceivables, Figure: "arOverdue",
	}}

	whole := boardCheck(want, shown(&board.Board{
		Blocks: []board.Block{records, metric}, Readings: []board.Reading{figure},
	}))
	assert.True(t, whole.Passed, whole.Detail)
	assert.Contains(t, whole.Name, "showing records of worker_credential")

	noFigure := boardCheck(want, shown(&board.Board{Blocks: []board.Block{records, metric}}))
	assert.False(t, noFigure.Passed)

	unmeasured := records
	unmeasured.Measure = ""
	noMeasure := boardCheck(want, shown(&board.Board{
		Blocks: []board.Block{unmeasured, metric}, Readings: []board.Reading{figure},
	}))
	assert.False(t, noMeasure.Passed)

	noKind := boardCheck(want, shown(&board.Board{
		Blocks: []board.Block{metric}, Readings: []board.Reading{figure},
	}))
	assert.False(t, noKind.Passed)
}

func linkedTurn(t *testing.T, made bool) *benchrecord.TurnRecord {
	t.Helper()

	draft := pulid.MustNew("art_")
	layout := &board.Board{
		Title:    "Lapsed credentials",
		Verdict:  "Jane Doe's card lapsed; I wrote [a reminder](artifact:" + draft.String() + ").",
		Place:    board.PlaceConversation,
		Readings: []board.Reading{{Key: "late"}},
	}
	payload, err := (&board.Payload{Board: *layout, Place: layout.Placed()}).Map()
	require.NoError(t, err)
	id := pulid.MustNew("art_")
	turn := &benchrecord.TurnRecord{
		Artifacts: []assistantartifactports.AssistantArtifact{{
			ID: id, Kind: assistantartifact.KindBoard, Title: layout.Title, Payload: payload,
		}},
	}
	turn.Reply = boardReply(id)
	if made {
		turn.Artifacts = append(turn.Artifacts, assistantartifactports.AssistantArtifact{
			ID: draft, Kind: assistantartifact.KindDocument, Title: "Renewal reminder",
		})
	}

	return turn
}

func TestBoardCheck_PassesWordsThatLinkAnArtifactTheTurnMadeAndNoOther(t *testing.T) {
	t.Parallel()

	want := &benchscenario.ExpectedBoard{Place: "conversation", Links: true}

	check := boardCheck(want, []*benchrecord.TurnRecord{linkedTurn(t, true)})
	assert.True(t, check.Passed, check.Detail)
	assert.Contains(t, check.Name, "its words linking an artifact the turn made")

	assert.False(t, boardCheck(want, []*benchrecord.TurnRecord{linkedTurn(t, false)}).Passed)
	assert.False(t, boardCheck(&benchscenario.ExpectedBoard{Place: "conversation"},
		[]*benchrecord.TurnRecord{linkedTurn(t, false)}).Passed,
		"a link to an artifact the turn did not make fails every board check")
}
