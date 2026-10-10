package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The board's urgency follows the pickup window, so a load already picked up
reads Late whether or not it is behind on delivery. Asked for "the late
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
			BoardMove: &repositories.BoardMove{
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
	for _, row := range view.Moves {
		lateTo[row.ProNumber] = row.LateTo
	}
	assert.Equal(t, "pick up", lateTo["SEED-NOT-PICKED"])
	assert.Empty(t, lateTo["SEED-ON-TIME"], "picked up and inside its delivery window is not late")
	assert.Equal(t, "deliver", lateTo["SEED-LATE-DELIVERY"])
	assert.Contains(t, view.Note, "SEED-LATE-DELIVERY", "the note names every Delayed shipment")
	assert.Contains(t, view.Note, "1 late to deliver")
	assert.Contains(t, view.Note, "1 late to pick up")
	assert.Equal(t, "SEED-LATE-DELIVERY", view.Moves[0].ProNumber,
		"a load late to deliver comes first, so shortening a long board cannot drop it")
	assert.Equal(t, "SEED-NOT-PICKED", view.Moves[1].ProNumber)
}
