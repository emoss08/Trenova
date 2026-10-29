package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestShipment_ShipperStop(t *testing.T) {
	t.Parallel()

	latePickup := &Stop{
		ID:       pulid.ID("stp_late"),
		Type:     shipmentvocab.StopTypePickup,
		Sequence: 5,
	}
	firstPickup := &Stop{
		ID:       pulid.ID("stp_first"),
		Type:     shipmentvocab.StopTypeSplitPickup,
		Sequence: 2,
	}
	delivery := &Stop{
		ID:       pulid.ID("stp_delivery"),
		Type:     shipmentvocab.StopTypeDelivery,
		Sequence: 0,
	}
	earlierMovePickup := &Stop{
		ID:       pulid.ID("stp_earlier"),
		Type:     shipmentvocab.StopTypePickup,
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
					{
						ID:       pulid.ID("stp_delivery"),
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 1,
					},
				},
			},
		},
	}

	if got := entity.ShipperStop(); got != nil {
		t.Fatalf("ShipperStop() = %v, want nil", got)
	}
}

func TestShipment_PayerID(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	billTo := pulid.MustNew("cus_")

	require.Equal(t, customerID, (&Shipment{CustomerID: customerID}).PayerID())
	require.Equal(
		t,
		billTo,
		(&Shipment{CustomerID: customerID, BillToCustomerID: &billTo}).PayerID(),
	)
	nilID := pulid.Nil
	require.Equal(
		t,
		customerID,
		(&Shipment{CustomerID: customerID, BillToCustomerID: &nilID}).PayerID(),
	)
	var none *Shipment
	require.True(t, none.PayerID().IsNil())
	require.False(t, none.HasExplicitBillTo())
	require.False(t, (&Shipment{CustomerID: customerID}).HasExplicitBillTo())
	require.True(
		t,
		(&Shipment{CustomerID: customerID, BillToCustomerID: &billTo}).HasExplicitBillTo(),
	)
}
