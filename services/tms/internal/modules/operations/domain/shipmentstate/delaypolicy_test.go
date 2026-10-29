package shipmentstate

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/stretchr/testify/assert"
)

func TestIsDelayedEligibleShipmentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status shipmentvocab.Status
		want   bool
	}{
		{name: "new is eligible", status: shipmentvocab.StatusNew, want: true},
		{name: "in transit is eligible", status: shipmentvocab.StatusInTransit, want: true},
		{name: "delayed is not eligible", status: shipmentvocab.StatusDelayed, want: false},
		{
			name:   "ready to invoice is not eligible",
			status: shipmentvocab.StatusReadyToInvoice,
			want:   false,
		},
		{name: "completed is not eligible", status: shipmentvocab.StatusCompleted, want: false},
		{name: "invoiced is not eligible", status: shipmentvocab.StatusInvoiced, want: false},
		{name: "canceled is not eligible", status: shipmentvocab.StatusCanceled, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, IsDelayedEligibleShipmentStatus(tt.status))
		})
	}
}

func TestIsStopOverdue_UsesCurrentTimeAndScheduledCutoff(t *testing.T) {
	t.Parallel()

	stop := &shipment.Stop{
		Status:               shipmentvocab.StopStatusInTransit,
		ScheduledWindowStart: 100,
		ScheduledWindowEnd:   new(int64(200)),
	}

	assert.False(t, IsStopOverdue(stop, 200+(30*60), 30))
	assert.True(t, IsStopOverdue(stop, 201+(30*60), 30))
	assert.False(t, IsStopOverdue(stop, 201+(30*60), DisabledDelayThresholdMinutes))
}

func TestIsStopOverdue_FallsBackToScheduledStart(t *testing.T) {
	t.Parallel()

	stop := &shipment.Stop{
		Status:               shipmentvocab.StopStatusInTransit,
		ScheduledWindowStart: 100,
	}

	assert.False(t, IsStopOverdue(stop, 100+(30*60), 30))
	assert.True(t, IsStopOverdue(stop, 101+(30*60), 30))
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}
