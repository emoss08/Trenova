package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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

// Clearing an override is a deliberate act, never a side effect of saving
// something else. A client that round-trips a shipment without the rating
// fields must not thereby unlock an invoiced load or wipe a negotiated rate.
func TestRestoreRateOwnedFields_CallersCannotClearAnOverride(t *testing.T) {
	t.Parallel()

	setBy := pulid.MustNew("usr_")
	setAt := int64(500)
	quoteID := pulid.MustNew("rqt_")
	agreementID := pulid.MustNew("ragr_")
	ruleID := pulid.MustNew("ragr_")

	original := &Shipment{
		RateOverrideAmount:  decimal.NewNullDecimal(decimal.NewFromInt(1400)),
		RateOverrideReason:  "Spot rate agreed with the customer",
		RateOverrideByID:    &setBy,
		RateOverrideAt:      &setAt,
		RateLocked:          true,
		RateQuoteID:         &quoteID,
		RateAgreementID:     &agreementID,
		RateAgreementRuleID: &ruleID,
	}

	updated := &Shipment{}

	RestoreRateOwnedFields(original, updated)

	require.True(t, updated.HasRateOverride())
	require.True(t, decimal.NewFromInt(1400).Equal(updated.RateOverrideAmount.Decimal))
	require.Equal(t, "Spot rate agreed with the customer", updated.RateOverrideReason)
	require.Equal(t, setBy, *updated.RateOverrideByID)
	require.Equal(t, setAt, *updated.RateOverrideAt)
	require.True(t, updated.RateLocked)
	require.Equal(t, quoteID, *updated.RateQuoteID)
	require.Equal(t, agreementID, *updated.RateAgreementID)
	require.Equal(t, ruleID, *updated.RateAgreementRuleID)
}

// A forged override on a create — where there is no original to compare
// against — is left alone here; the create path never calls this.
func TestRestoreRateOwnedFields_IgnoresAMissingSide(t *testing.T) {
	t.Parallel()

	updated := &Shipment{RateLocked: true}

	RestoreRateOwnedFields(nil, updated)
	RestoreRateOwnedFields(&Shipment{}, nil)

	require.True(t, updated.RateLocked)
}
