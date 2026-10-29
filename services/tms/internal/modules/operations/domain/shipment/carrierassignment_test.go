package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
)

func TestCarrierAssignment_CancelRecordsWhenAndWhy(t *testing.T) {
	t.Parallel()

	entity := &CarrierAssignment{Status: shipmentvocab.CarrierAssignmentStatusConfirmed}
	entity.Cancel(1790000000, "Carrier fell off")

	assert.Equal(t, shipmentvocab.CarrierAssignmentStatusCanceled, entity.Status)
	assert.Equal(t, int64(1790000000), *entity.CanceledAt)
	assert.Equal(t, "Carrier fell off", entity.CancellationReason)
	assert.False(t, entity.IsActive())
}

// Only a pending assignment is confirmed, and only a confirmed one reverts:
// each reports whether it moved, so a caller writes only what changed.
func TestCarrierAssignment_ConfirmationMovesOnlyFromTheStateItExpects(t *testing.T) {
	t.Parallel()

	pending := &CarrierAssignment{Status: shipmentvocab.CarrierAssignmentStatusPending}
	assert.True(t, pending.Confirm(1790000000))
	assert.Equal(t, shipmentvocab.CarrierAssignmentStatusConfirmed, pending.Status)
	assert.Equal(t, int64(1790000000), *pending.ConfirmedAt)
	assert.False(t, pending.Confirm(1790000500), "a confirmed assignment stays as it is")
	assert.Equal(t, int64(1790000000), *pending.ConfirmedAt)

	assert.True(t, pending.RevertConfirmation())
	assert.Equal(t, shipmentvocab.CarrierAssignmentStatusPending, pending.Status)
	assert.Nil(t, pending.ConfirmedAt)
	assert.False(t, pending.RevertConfirmation(), "a pending assignment has nothing to revert")

	canceled := &CarrierAssignment{Status: shipmentvocab.CarrierAssignmentStatusCanceled}
	assert.False(t, canceled.Confirm(1790000000))
	assert.Equal(t, shipmentvocab.CarrierAssignmentStatusCanceled, canceled.Status)
}
