package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The board's urgency once followed the pickup window alone, so a load already
picked up read Late whether or not it was behind on delivery; it now follows
the delivery once the load is picked up. Asked for "the late
freshhaul reefer", Haiku saw SEED-SHP-010, on the road and past its delivery
window, beside ten loads nobody had picked up, all Late, and asked which was
meant; asked what is running late, gpt-6-luna listed 40 of 78 Late moves and
left out the one load marked Delayed. Each late move says what it is late to
do, and the note names every Delayed shipment, where list shortening cannot
reach it.
*/
func TestBoardView_SaysWhatEachLateMoveIsLateFor(t *testing.T) {
	t.Parallel()

	const now = int64(1_790_000_000)
	past, future := now-3600, now+3600
	arrived := now - 86_400
	move := func(pro string, status shipment.Status, originArrive *int64, destEnd int64) *dispatchconsoleservice.BoardMove {
		return &dispatchconsoleservice.BoardMove{
			BoardMove: &shipmentports.BoardMove{
				MoveID:                 pulid.MustNew("sm_"),
				ShipmentID:             pulid.MustNew("shp_"),
				ProNumber:              pro,
				ShipmentStatus:         status,
				OriginWindowStart:      now - 2*86_400,
				OriginActualArrive:     originArrive,
				DestinationWindowStart: destEnd - 7200,
				DestinationWindowEnd:   &destEnd,
			},
			Urgency: dispatchconsoleservice.UrgencyLate,
		}
	}
	board := &dispatchconsoleservice.Board{
		WindowStart: now,
		WindowEnd:   now + 86_400,
		Moves: []*dispatchconsoleservice.BoardMove{
			move("SEED-NOT-PICKED", shipment.StatusNew, nil, future),
			move("SEED-ON-TIME", shipment.StatusInTransit, &arrived, future),
			move("SEED-LATE-DELIVERY", shipment.StatusDelayed, &arrived, past),
		},
	}

	view := boardViewOf(board, "UTC", defaultBoardLimit)
	view.tally()

	require.Len(t, view.Moves, 3)
	lateTo := map[string]string{}
	lateBy := map[string]string{}
	for _, row := range view.Moves {
		lateTo[row.ProNumber] = row.LateTo
		lateBy[row.ProNumber] = row.LateBy
	}
	assert.Equal(t, "pick up", lateTo["SEED-NOT-PICKED"])
	assert.Empty(t, lateTo["SEED-ON-TIME"], "picked up and inside its delivery window is not late")
	assert.Equal(t, "deliver", lateTo["SEED-LATE-DELIVERY"])
	assert.Equal(t, "2d 0h", lateBy["SEED-NOT-PICKED"],
		"lateness reads as a person says it, not in minutes")
	assert.Equal(t, "1h 00m", lateBy["SEED-LATE-DELIVERY"])
	assert.Empty(t, lateBy["SEED-ON-TIME"])
	assert.Contains(t, view.Note, "SEED-LATE-DELIVERY", "the note names every Delayed shipment")
	assert.Contains(t, view.Note, "1 late to deliver")
	assert.Contains(t, view.Note, "1 late to pick up")
}

/*
Asked about a late reefer, Haiku repeated lateMinutes and replied "1,320
minutes late". Each late move carries how long past the window's close it
is as a person says it, and a pickup window still open has none yet.
*/
func TestLateness_MeasuresFromTheWindowsClose(t *testing.T) {
	t.Parallel()

	const now = int64(1_790_000_000)
	end := now + 1800
	open := &shipmentports.BoardMove{OriginWindowStart: now - 1800, OriginWindowEnd: &end}
	what, seconds := lateness(open, now)
	assert.Equal(t, "pick up", what)
	assert.Empty(t, lateFor(seconds), "the pickup window is still open")

	closed := now - 22*3600
	overdue := &shipmentports.BoardMove{OriginWindowStart: closed - 7200, OriginWindowEnd: &closed}
	what, seconds = lateness(overdue, now)
	assert.Equal(t, "pick up", what)
	assert.Equal(t, "22h 00m", lateFor(seconds))

	arrived := now - 5*86_400
	deadline := now - 3*86_400
	delivering := &shipmentports.BoardMove{
		OriginWindowStart:      arrived,
		OriginActualArrive:     &arrived,
		DestinationWindowStart: deadline - 3600,
		DestinationWindowEnd:   &deadline,
	}
	what, seconds = lateness(delivering, now)
	assert.Equal(t, "deliver", what)
	assert.Equal(t, "3d 0h", lateFor(seconds))
}

/*
Asked for the operations board, Haiku named the first late move "the most
overdue": it was three hours behind, with others fourteen hours behind further
down, because late moves kept the board's own order. The board is now read
due-first, the furthest behind leading, a page at a time; the view keeps that
order, so a page cut short resumes where it ended.
*/
func TestDispatchBoard_ReadsDueFirstAndPagesTheLateMoves(t *testing.T) {
	t.Parallel()

	const now = int64(1_790_000_000)
	arrived := now - 5*86_400
	delivering := func(pro string, behind int64) *dispatchconsoleservice.BoardMove {
		deadline := now - behind
		return &dispatchconsoleservice.BoardMove{
			BoardMove: &shipmentports.BoardMove{
				MoveID:                 pulid.MustNew("sm_"),
				ShipmentID:             pulid.MustNew("shp_"),
				ProNumber:              pro,
				ShipmentStatus:         shipment.StatusDelayed,
				OriginWindowStart:      arrived,
				OriginActualArrive:     &arrived,
				DestinationWindowStart: deadline - 3600,
				DestinationWindowEnd:   &deadline,
			},
			Urgency: dispatchconsoleservice.UrgencyLate,
		}
	}
	reader := &fakeBoard{board: &dispatchconsoleservice.Board{
		WindowStart: now,
		WindowEnd:   now + 86_400,
		Moves: []*dispatchconsoleservice.BoardMove{
			delivering("FOURTEEN-HOURS", 14*3600),
			delivering("ELEVEN-HOURS", 11*3600),
			delivering("THREE-HOURS", 3*3600),
		},
		Drivers: []*dispatchconsoleservice.BoardDriver{},
	}}
	tool := newGetDispatchBoardTool(reader)

	call := keyedCall(map[string]any{paramBoardLate: true, paramLimit: 2})
	data, err := tool.Query(t.Context(), call)
	require.NoError(t, err)

	sent := reader.sent
	require.NotNil(t, sent)
	assert.True(t, sent.LateOnly)
	assert.True(t, sent.DueFirst, "the board is read furthest behind first")
	assert.Equal(t, 3, sent.Limit, "one more than the page, to know there is another")
	assert.Zero(t, sent.Offset)

	view, ok := data.(*boardView)
	require.True(t, ok)
	order := make([]string, 0, len(view.Moves))
	for _, row := range view.Moves {
		order = append(order, row.ProNumber)
	}
	assert.Equal(t, []string{"FOURTEEN-HOURS", "ELEVEN-HOURS"}, order)
	assert.Equal(t, "14h 00m", view.Moves[0].LateBy)
	assert.Nil(t, view.Drivers, "a question about late loads leaves the drivers out")
	assert.True(t, view.HasMore)
	assert.Contains(t, view.Note, "2 late moves listed")

	_, err = tool.Query(t.Context(), keyedCall(map[string]any{
		paramBoardLate: true, paramLimit: 2, paramAfter: view.NextCursor,
	}))
	require.NoError(t, err)
	assert.Equal(t, 2, reader.sent.Offset, "the next page starts after the two read")

	cut, ok := view.CutPage(1)
	require.True(t, ok)
	short, _ := cut.(*boardView)
	assert.Len(t, short.Moves, 1)
	assert.True(t, short.HasMore)
	scope, err := cursorScope(tool.Name(), call.Params)
	require.NoError(t, err)
	key, err := pagination.DecodeKeyCursor(scope, short.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "1", key, "a cut board resumes at the move after the last one kept")
	assert.Contains(t, short.Note, "1 late moves listed")
}

type fakeBoard struct {
	board *dispatchconsoleservice.Board
	sent  *dispatchconsoleservice.GetBoardRequest
}

func (f *fakeBoard) GetBoard(
	_ context.Context,
	req *dispatchconsoleservice.GetBoardRequest,
) (*dispatchconsoleservice.Board, error) {
	f.sent = req

	return f.board, nil
}
