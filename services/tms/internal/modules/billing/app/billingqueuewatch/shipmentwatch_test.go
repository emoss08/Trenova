package billingqueuewatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/stretchr/testify/assert"
)

func TestShipmentChangesThatCanMoveABillRecheckIt(t *testing.T) {
	t.Parallel()

	for _, eventType := range []shipmentevent.Type{
		shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeStopDeparted,
		shipmentevent.TypeStopArrivalCorrected,
		shipmentevent.TypeStopDepartureCorrected,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldReleased,
	} {
		assert.True(t, RechecksOn(eventType), eventType)
	}
	for _, eventType := range []shipmentevent.Type{
		shipmentevent.TypeCommentPosted,
		shipmentevent.TypeTenderOffered,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeStopArrived,
	} {
		assert.False(t, RechecksOn(eventType), eventType)
	}
}
