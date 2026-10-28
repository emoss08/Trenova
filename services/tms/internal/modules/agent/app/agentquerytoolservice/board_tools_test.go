package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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

	view := boardViewOf(board, "UTC")

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
	assert.Equal(t, "SEED-LATE-DELIVERY", view.Moves[0].ProNumber,
		"a load late to deliver comes first, so shortening a long board cannot drop it")
	assert.Equal(t, "SEED-NOT-PICKED", view.Moves[1].ProNumber)
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
	what, by := lateness(open, now)
	assert.Equal(t, "pick up", what)
	assert.Empty(t, by, "the pickup window is still open")

	closed := now - 22*3600
	overdue := &shipmentports.BoardMove{OriginWindowStart: closed - 7200, OriginWindowEnd: &closed}
	what, by = lateness(overdue, now)
	assert.Equal(t, "pick up", what)
	assert.Equal(t, "22h 00m", by)

	arrived := now - 5*86_400
	deadline := now - 3*86_400
	delivering := &shipmentports.BoardMove{
		OriginWindowStart:      arrived,
		OriginActualArrive:     &arrived,
		DestinationWindowStart: deadline - 3600,
		DestinationWindowEnd:   &deadline,
	}
	what, by = lateness(delivering, now)
	assert.Equal(t, "deliver", what)
	assert.Equal(t, "3d 0h", by)
}
