package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACeilingRefusesWhateverTheOverageSetting(t *testing.T) {
	t.Parallel()

	shipments := platformcatalog.MeterShipmentsTotal
	tests := []struct {
		name      string
		meter     managedusage.Meter
		overage   bool
		requested int64
		allowed   bool
		reason    managedusage.Reason
	}{
		{
			name: "with overage, up to the ceiling",
			meter: managedusage.Meter{
				Meter:     shipments,
				Quantity:  1_400,
				Allowance: 100,
				Ceiling:   1_500,
			},
			overage:   true,
			requested: 100,
			allowed:   true,
			reason:    managedusage.ReasonWithin,
		},
		{
			name: "with overage, past the ceiling",
			meter: managedusage.Meter{
				Meter:     shipments,
				Quantity:  1_400,
				Allowance: 100,
				Ceiling:   1_500,
			},
			overage:   true,
			requested: 101,
			reason:    managedusage.ReasonCeiling,
		},
		{
			name: "already at the ceiling",
			meter: managedusage.Meter{
				Meter:     shipments,
				Quantity:  1_500,
				Allowance: 100,
				Ceiling:   1_500,
			},
			overage:   true,
			requested: 1,
			reason:    managedusage.ReasonCeiling,
		},
		{
			name:      "no ceiling, overage carries on",
			meter:     managedusage.Meter{Meter: shipments, Quantity: 1_000_000, Allowance: 100},
			overage:   true,
			requested: 1,
			allowed:   true,
			reason:    managedusage.ReasonWithin,
		},
		{
			name: "without overage the allowance stops it first",
			meter: managedusage.Meter{
				Meter:     shipments,
				Quantity:  100,
				Allowance: 100,
				Ceiling:   1_500,
			},
			requested: 1,
			reason:    managedusage.ReasonAllowance,
		},
		{
			name: "reading usage asks for nothing",
			meter: managedusage.Meter{
				Meter:     shipments,
				Quantity:  1_500,
				Allowance: 100,
				Ceiling:   1_500,
			},
			overage: true,
			allowed: true,
			reason:  managedusage.ReasonWithin,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := managedusage.Decide(state(tt.overage, tt.meter), shipments, tt.requested)
			assert.Equal(t, tt.allowed, d.Allowed)
			assert.Equal(t, tt.reason, d.Reason)
			assert.Equal(t, tt.reason == managedusage.ReasonCeiling, d.CeilingReached())
			assert.Equal(t, tt.meter.Ceiling, d.Ceiling)
		})
	}
}

func TestACeilingIsZeroOrAtLeastTheAllowance(t *testing.T) {
	t.Parallel()

	meter := func(ceiling int64) managedusage.State {
		return managedusage.State{Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterShipmentsTotal,
			Unit:      managedusage.UnitCount,
			Allowance: 100,
			Ceiling:   ceiling,
		}}}
	}

	for _, ok := range []int64{0, 100, 1_500} {
		s := meter(ok)
		require.NoError(t, s.Validate(), ok)
	}
	for _, bad := range []int64{-1, 99} {
		s := meter(bad)
		validationtest.Refused(t, s.Validate()).
			Exactly(validationtest.At("meters[0].ceiling", errortypes.ErrInvalid))
	}
}

func TestTheCeilingRefusalNamesTheCeilingAndNoPrice(t *testing.T) {
	t.Parallel()

	err := managedusage.CeilingRefusal(platformcatalog.MeterShipmentsTotal, 1_500, 1_500, "managed")
	assert.Equal(t, int64(1_500), err.Limit)
	assert.Equal(t, []any{int64(1_500)}, err.Args)
	assert.Contains(t, err.Message, "at most {0}")
	assert.True(t, errortypes.IsQuotaExceededError(err))
}
