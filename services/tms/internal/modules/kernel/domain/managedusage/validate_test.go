package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	meter := managedusage.Meter{
		Meter:     platformcatalog.MeterShipmentsTotal,
		Unit:      managedusage.UnitCount,
		Quantity:  1,
		Allowance: 10,
	}
	tests := []struct {
		name  string
		state managedusage.State
		ok    bool
	}{
		{
			name: "a window and a meter",
			state: managedusage.State{
				WindowStart: 1,
				WindowEnd:   2,
				Meters:      []managedusage.Meter{meter},
			},
			ok: true,
		},
		{name: "no window and no meters", state: managedusage.State{}, ok: true},
		{name: "half a window", state: managedusage.State{WindowStart: 1}},
		{name: "a window that ends first", state: managedusage.State{WindowStart: 2, WindowEnd: 1}},
		{
			name:  "a meter twice",
			state: managedusage.State{Meters: []managedusage.Meter{meter, meter}},
		},
		{
			name: "an unknown unit",
			state: managedusage.State{Meters: []managedusage.Meter{{
				Meter: platformcatalog.MeterShipmentsTotal, Unit: "minutes",
			}}},
		},
		{
			name: "a negative quantity",
			state: managedusage.State{Meters: []managedusage.Meter{{
				Meter: platformcatalog.MeterShipmentsTotal, Unit: managedusage.UnitCount, Quantity: -1,
			}}},
		},
		{
			name: "no meter key",
			state: managedusage.State{Meters: []managedusage.Meter{{
				Unit: managedusage.UnitCount,
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.state.Validate()
			if tt.ok {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}
