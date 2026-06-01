package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
)

func TestShipment_ApplyEntryMethodDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		shipment *Shipment
		original *Shipment
		want     EntryMethod
	}{
		{
			name:     "defaults create to manual",
			shipment: &Shipment{},
			want:     EntryMethodManual,
		},
		{
			name:     "preserves original update value",
			shipment: &Shipment{},
			original: &Shipment{EntryMethod: EntryMethodEDI},
			want:     EntryMethodEDI,
		},
		{
			name:     "keeps explicit update value",
			shipment: &Shipment{EntryMethod: EntryMethodManual},
			original: &Shipment{EntryMethod: EntryMethodEDI},
			want:     EntryMethodManual,
		},
		{
			name:     "leaves invalid explicit value for validation",
			shipment: &Shipment{EntryMethod: EntryMethod("CarrierPortal")},
			want:     EntryMethod("CarrierPortal"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.shipment.ApplyEntryMethodDefault(tt.original)

			if tt.shipment.EntryMethod != tt.want {
				t.Fatalf("EntryMethod = %q, want %q", tt.shipment.EntryMethod, tt.want)
			}
		})
	}
}

func TestShipment_ShipperStop(t *testing.T) {
	t.Parallel()

	latePickup := &Stop{
		ID:       pulid.ID("stp_late"),
		Type:     StopTypePickup,
		Sequence: 5,
	}
	firstPickup := &Stop{
		ID:       pulid.ID("stp_first"),
		Type:     StopTypeSplitPickup,
		Sequence: 2,
	}
	delivery := &Stop{
		ID:       pulid.ID("stp_delivery"),
		Type:     StopTypeDelivery,
		Sequence: 0,
	}
	earlierMovePickup := &Stop{
		ID:       pulid.ID("stp_earlier"),
		Type:     StopTypePickup,
		Sequence: 8,
	}
	entity := &Shipment{
		Moves: []*ShipmentMove{
			{
				Sequence: 3,
				Stops:    []*Stop{delivery, firstPickup, latePickup},
			},
			nil,
			{
				Sequence: 1,
				Stops:    []*Stop{earlierMovePickup},
			},
		},
	}

	if got := entity.ShipperStop(); got != earlierMovePickup {
		t.Fatalf("ShipperStop() = %v, want %v", got, earlierMovePickup)
	}
}

func TestShipment_ShipperStopSkipsNilAndNonOriginStops(t *testing.T) {
	t.Parallel()

	entity := &Shipment{
		Moves: []*ShipmentMove{
			nil,
			{
				Sequence: 1,
				Stops: []*Stop{
					nil,
					{ID: pulid.ID("stp_delivery"), Type: StopTypeDelivery, Sequence: 1},
				},
			},
		},
	}

	if got := entity.ShipperStop(); got != nil {
		t.Fatalf("ShipperStop() = %v, want nil", got)
	}
}
