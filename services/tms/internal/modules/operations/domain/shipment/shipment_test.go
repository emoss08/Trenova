package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
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

func TestShipment_NormalizeBillTo(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")

	same := customerID
	entity := &Shipment{CustomerID: customerID, BillToCustomerID: &same}
	entity.NormalizeBillTo()
	require.Nil(t, entity.BillToCustomerID, "a bill-to naming the customer is the same as none")

	empty := pulid.Nil
	entity = &Shipment{CustomerID: customerID, BillToCustomerID: &empty}
	entity.NormalizeBillTo()
	require.Nil(t, entity.BillToCustomerID)

	entity = &Shipment{CustomerID: customerID, BillToCustomerID: &other}
	entity.NormalizeBillTo()
	require.NotNil(t, entity.BillToCustomerID)
	require.Equal(t, other, *entity.BillToCustomerID)

	var none *Shipment
	none.NormalizeBillTo()
}

func TestFollowCustomerChange(t *testing.T) {
	t.Parallel()

	oldCustomer := pulid.MustNew("cus_")
	newCustomer := pulid.MustNew("cus_")
	thirdParty := pulid.MustNew("cus_")

	tests := []struct {
		name     string
		original *Shipment
		updated  *Shipment
		want     *pulid.ID
	}{
		{
			name:     "bill-to that named the old customer follows the new one",
			original: &Shipment{CustomerID: oldCustomer},
			updated:  &Shipment{CustomerID: newCustomer, BillToCustomerID: &oldCustomer},
			want:     nil,
		},
		{
			name:     "a third-party bill-to is left alone",
			original: &Shipment{CustomerID: oldCustomer},
			updated:  &Shipment{CustomerID: newCustomer, BillToCustomerID: &thirdParty},
			want:     &thirdParty,
		},
		{
			name:     "no-op when the customer did not change",
			original: &Shipment{CustomerID: oldCustomer},
			updated:  &Shipment{CustomerID: oldCustomer, BillToCustomerID: &oldCustomer},
			want:     &oldCustomer,
		},
		{
			name:     "no-op without an original",
			original: nil,
			updated:  &Shipment{CustomerID: newCustomer, BillToCustomerID: &oldCustomer},
			want:     &oldCustomer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			FollowCustomerChange(tc.original, tc.updated)
			if tc.want == nil {
				require.Nil(t, tc.updated.BillToCustomerID)
				return
			}
			require.NotNil(t, tc.updated.BillToCustomerID)
			require.Equal(t, *tc.want, *tc.updated.BillToCustomerID)
		})
	}
}

func TestShipment_ApplyFreightTermsDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		shipment *Shipment
		original *Shipment
		want     FreightTerms
	}{
		{name: "defaults to prepaid", shipment: &Shipment{}, want: FreightTermsPrepaid},
		{
			name:     "preserves the original on update",
			shipment: &Shipment{},
			original: &Shipment{FreightTerms: FreightTermsCollect},
			want:     FreightTermsCollect,
		},
		{
			name:     "keeps an explicit value",
			shipment: &Shipment{FreightTerms: FreightTermsThirdParty},
			original: &Shipment{FreightTerms: FreightTermsCollect},
			want:     FreightTermsThirdParty,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.shipment.ApplyFreightTermsDefault(tc.original)
			require.Equal(t, tc.want, tc.shipment.FreightTerms)
		})
	}

	var none *Shipment
	none.ApplyFreightTermsDefault(nil)
}

func TestFreightTerms_IsValid(t *testing.T) {
	t.Parallel()

	require.True(t, FreightTermsPrepaid.IsValid())
	require.True(t, FreightTermsCollect.IsValid())
	require.True(t, FreightTermsThirdParty.IsValid())
	require.False(t, FreightTerms("").IsValid())
	require.False(t, FreightTerms("Consignee").IsValid())
}
