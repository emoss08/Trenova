package shipment

import "testing"

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
