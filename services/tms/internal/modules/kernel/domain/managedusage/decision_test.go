package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func state(overage bool, m managedusage.Meter) *managedusage.State {
	return &managedusage.State{
		ID:          managedusage.CurrentStateID,
		WindowStart: 1_700_000_000,
		WindowEnd:   1_702_592_000,
		Overage:     overage,
		Meters:      []managedusage.Meter{m},
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()

	shipments := platformcatalog.MeterShipmentsTotal
	within := managedusage.Meter{Meter: shipments, Quantity: 90, Allowance: 100}
	noWindow := state(true, within)
	noWindow.WindowStart, noWindow.WindowEnd = 0, 0

	tests := []struct {
		name      string
		state     *managedusage.State
		meter     platformcatalog.MeterKey
		requested int64
		allowed   bool
		managed   bool
		reason    managedusage.Reason
	}{
		{
			name:    "no state yet",
			meter:   shipments,
			allowed: true,
			reason:  managedusage.ReasonNotManaged,
		},
		{
			name:    "a meter the control plane does not limit",
			state:   state(false, within),
			meter:   platformcatalog.MeterCustomersTotal,
			allowed: true,
			reason:  managedusage.ReasonNotManaged,
		},
		{
			name:      "within the allowance",
			state:     state(false, within),
			meter:     shipments,
			requested: 10,
			allowed:   true,
			managed:   true,
			reason:    managedusage.ReasonWithin,
		},
		{
			name:      "a batch that would cross the allowance without overage",
			state:     state(false, within),
			meter:     shipments,
			requested: 11,
			managed:   true,
			reason:    managedusage.ReasonAllowance,
		},
		{
			name:      "past the allowance with overage billed",
			state:     state(true, within),
			meter:     shipments,
			requested: 50,
			allowed:   true,
			managed:   true,
			reason:    managedusage.ReasonWithin,
		},
		{
			name: "a hard cap holds even with overage billed",
			state: state(true, managedusage.Meter{
				Meter: shipments, Quantity: 90, Allowance: 100, HardCap: true,
			}),
			meter:     shipments,
			requested: 11,
			managed:   true,
			reason:    managedusage.ReasonAllowance,
		},
		{
			name: "stopped by the control plane",
			state: state(true, managedusage.Meter{
				Meter: shipments, Quantity: 10, Allowance: 100, Stopped: true,
			}),
			meter:     shipments,
			requested: 1,
			managed:   true,
			reason:    managedusage.ReasonStopped,
		},
		{
			name:      "no open window stops every named meter",
			state:     noWindow,
			meter:     shipments,
			requested: 1,
			managed:   true,
			reason:    managedusage.ReasonNoWindow,
		},
		{
			name: "a check of nothing passes at the allowance",
			state: state(false, managedusage.Meter{
				Meter: shipments, Quantity: 100, Allowance: 100,
			}),
			meter:   shipments,
			allowed: true,
			managed: true,
			reason:  managedusage.ReasonWithin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := managedusage.Decide(tt.state, tt.meter, tt.requested)
			assert.Equal(t, tt.allowed, d.Allowed)
			assert.Equal(t, tt.managed, d.Managed)
			assert.Equal(t, tt.reason, d.Reason)
		})
	}
}

func TestRemaining(t *testing.T) {
	t.Parallel()

	d := managedusage.Decide(state(true, managedusage.Meter{
		Meter: platformcatalog.MeterShipmentsTotal, Quantity: 40, Allowance: 100,
	}), platformcatalog.MeterShipmentsTotal, 1)
	assert.Equal(t, int64(60), d.Remaining())

	d = managedusage.Decide(state(true, managedusage.Meter{
		Meter: platformcatalog.MeterShipmentsTotal, Quantity: 140, Allowance: 100,
	}), platformcatalog.MeterShipmentsTotal, 1)
	assert.Zero(t, d.Remaining())
}
