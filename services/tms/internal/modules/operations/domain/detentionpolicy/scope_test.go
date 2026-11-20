package detentionpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCoversAllFreight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*detentionpolicy.DetentionPolicy)
		want   bool
	}{
		{name: "no scope", mutate: func(*detentionpolicy.DetentionPolicy) {}, want: true},
		{
			name:   "customer",
			mutate: func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = new(pulid.MustNew("cus_")) },
		},
		{
			name:   "facility",
			mutate: func(p *detentionpolicy.DetentionPolicy) { p.LocationID = new(pulid.MustNew("loc_")) },
		},
		{
			name: "stop type",
			mutate: func(p *detentionpolicy.DetentionPolicy) {
				p.StopTypes = []shipmentvocab.StopType{shipmentvocab.StopTypePickup}
			},
		},
		{
			name:   "appointment stops only",
			mutate: func(p *detentionpolicy.DetentionPolicy) { p.AppointmentStopsOnly = true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := validPolicy()
			tt.mutate(p)

			assert.Equal(t, tt.want, p.CoversAllFreight())
		})
	}
}

func TestSameScope(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	typeA := pulid.MustNew("spt_")
	typeB := pulid.MustNew("spt_")

	tests := []struct {
		name  string
		left  func(*detentionpolicy.DetentionPolicy)
		right func(*detentionpolicy.DetentionPolicy)
		want  bool
	}{
		{
			name:  "both unscoped",
			left:  func(*detentionpolicy.DetentionPolicy) {},
			right: func(*detentionpolicy.DetentionPolicy) {},
			want:  true,
		},
		{
			name:  "nil and empty customer are the same",
			left:  func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = new(pulid.Nil) },
			right: func(*detentionpolicy.DetentionPolicy) {},
			want:  true,
		},
		{
			name:  "same customer",
			left:  func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = &customerID },
			right: func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = new(customerID) },
			want:  true,
		},
		{
			name:  "customer against unscoped",
			left:  func(p *detentionpolicy.DetentionPolicy) { p.CustomerID = &customerID },
			right: func(*detentionpolicy.DetentionPolicy) {},
		},
		{
			name: "shipment types in another order",
			left: func(p *detentionpolicy.DetentionPolicy) {
				p.ShipmentTypeIDs = []pulid.ID{typeA, typeB}
			},
			right: func(p *detentionpolicy.DetentionPolicy) {
				p.ShipmentTypeIDs = []pulid.ID{typeB, typeA}
			},
			want: true,
		},
		{
			name: "all freight against one shipment type",
			left: func(*detentionpolicy.DetentionPolicy) {},
			right: func(p *detentionpolicy.DetentionPolicy) {
				p.ShipmentTypeIDs = []pulid.ID{typeA}
			},
		},
		{
			name:  "appointment stops only differs",
			left:  func(*detentionpolicy.DetentionPolicy) {},
			right: func(p *detentionpolicy.DetentionPolicy) { p.AppointmentStopsOnly = true },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			left := validPolicy()
			tt.left(left)
			right := validPolicy()
			tt.right(right)

			assert.Equal(t, tt.want, left.SameScope(right))
			assert.Equal(t, tt.want, right.SameScope(left))
		})
	}
}
