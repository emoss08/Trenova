package carrierintelenrollment_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/stretchr/testify/assert"
)

func TestEnrollmentStateMachine(t *testing.T) {
	t.Parallel()

	e := &carrierintelenrollment.CarrierMonitoringEnrollment{
		DesiredState: carrierintelenrollment.DesiredStateNotEnrolled,
		VendorState:  carrierintelenrollment.VendorStateUnknown,
	}
	assert.True(
		t,
		e.MarkDesired(
			carrierintelenrollment.DesiredStateEnrolled,
			carrierintelenrollment.EnrollmentReasonManual,
			partiestest.CarrierIntelNow,
		),
	)
	assert.Equal(t, carrierintelenrollment.VendorStatePendingAdd, e.VendorState)
	assert.True(t, e.NeedsSync())

	e.MarkSynced(partiestest.CarrierIntelNow)
	assert.True(t, e.IsActive())
	assert.True(t, e.OwnedByTrenova)
	assert.False(t, e.NeedsSync())

	assert.False(
		t,
		e.MarkDesired(
			carrierintelenrollment.DesiredStateEnrolled,
			carrierintelenrollment.EnrollmentReasonManual,
			partiestest.CarrierIntelNow,
		),
	)
	assert.True(
		t,
		e.MarkDesired(
			carrierintelenrollment.DesiredStateNotEnrolled,
			"",
			partiestest.CarrierIntelNow,
		),
	)
	assert.Equal(t, carrierintelenrollment.VendorStatePendingRemove, e.VendorState)
	assert.True(t, e.NeedsSync())
	e.MarkSynced(partiestest.CarrierIntelNow)
	assert.Equal(t, carrierintelenrollment.VendorStateRemoved, e.VendorState)

	for range 5 {
		e.MarkFailed("boom", partiestest.CarrierIntelNow)
	}
	assert.Equal(t, carrierintelenrollment.VendorStateFailed, e.VendorState)
}
