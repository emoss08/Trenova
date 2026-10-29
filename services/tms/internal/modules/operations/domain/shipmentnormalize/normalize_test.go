package shipmentnormalize

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestShipment_ApplyEntryMethodDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		shipment *shipment.Shipment
		original *shipment.Shipment
		want     shipmentvocab.EntryMethod
	}{
		{
			name:     "defaults create to manual",
			shipment: &shipment.Shipment{},
			want:     shipmentvocab.EntryMethodManual,
		},
		{
			name:     "preserves original update value",
			shipment: &shipment.Shipment{},
			original: &shipment.Shipment{EntryMethod: shipmentvocab.EntryMethodEDI},
			want:     shipmentvocab.EntryMethodEDI,
		},
		{
			name:     "keeps explicit update value",
			shipment: &shipment.Shipment{EntryMethod: shipmentvocab.EntryMethodManual},
			original: &shipment.Shipment{EntryMethod: shipmentvocab.EntryMethodEDI},
			want:     shipmentvocab.EntryMethodManual,
		},
		{
			name:     "leaves invalid explicit value for validation",
			shipment: &shipment.Shipment{EntryMethod: shipmentvocab.EntryMethod("CarrierPortal")},
			want:     shipmentvocab.EntryMethod("CarrierPortal"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			EntryMethod(tt.shipment, tt.original)

			if tt.shipment.EntryMethod != tt.want {
				t.Fatalf("EntryMethod = %q, want %q", tt.shipment.EntryMethod, tt.want)
			}
		})
	}
}

func TestShipment_NormalizeBillTo(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")

	same := customerID
	entity := &shipment.Shipment{CustomerID: customerID, BillToCustomerID: &same}
	BillTo(entity)
	require.Nil(t, entity.BillToCustomerID, "a bill-to naming the customer is the same as none")

	empty := pulid.Nil
	entity = &shipment.Shipment{CustomerID: customerID, BillToCustomerID: &empty}
	BillTo(entity)
	require.Nil(t, entity.BillToCustomerID)

	entity = &shipment.Shipment{CustomerID: customerID, BillToCustomerID: &other}
	BillTo(entity)
	require.NotNil(t, entity.BillToCustomerID)
	require.Equal(t, other, *entity.BillToCustomerID)

	var none *shipment.Shipment
	BillTo(none)
}

func TestFollowCustomerChange(t *testing.T) {
	t.Parallel()

	oldCustomer := pulid.MustNew("cus_")
	newCustomer := pulid.MustNew("cus_")
	thirdParty := pulid.MustNew("cus_")

	tests := []struct {
		name     string
		original *shipment.Shipment
		updated  *shipment.Shipment
		want     *pulid.ID
	}{
		{
			name:     "bill-to that named the old customer follows the new one",
			original: &shipment.Shipment{CustomerID: oldCustomer},
			updated:  &shipment.Shipment{CustomerID: newCustomer, BillToCustomerID: &oldCustomer},
			want:     nil,
		},
		{
			name:     "a third-party bill-to is left alone",
			original: &shipment.Shipment{CustomerID: oldCustomer},
			updated:  &shipment.Shipment{CustomerID: newCustomer, BillToCustomerID: &thirdParty},
			want:     &thirdParty,
		},
		{
			name:     "no-op when the customer did not change",
			original: &shipment.Shipment{CustomerID: oldCustomer},
			updated:  &shipment.Shipment{CustomerID: oldCustomer, BillToCustomerID: &oldCustomer},
			want:     &oldCustomer,
		},
		{
			name:     "no-op without an original",
			original: nil,
			updated:  &shipment.Shipment{CustomerID: newCustomer, BillToCustomerID: &oldCustomer},
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
		shipment *shipment.Shipment
		original *shipment.Shipment
		want     shipmentvocab.FreightTerms
	}{
		{
			name:     "defaults to prepaid",
			shipment: &shipment.Shipment{},
			want:     shipmentvocab.FreightTermsPrepaid,
		},
		{
			name:     "preserves the original on update",
			shipment: &shipment.Shipment{},
			original: &shipment.Shipment{FreightTerms: shipmentvocab.FreightTermsCollect},
			want:     shipmentvocab.FreightTermsCollect,
		},
		{
			name:     "keeps an explicit value",
			shipment: &shipment.Shipment{FreightTerms: shipmentvocab.FreightTermsThirdParty},
			original: &shipment.Shipment{FreightTerms: shipmentvocab.FreightTermsCollect},
			want:     shipmentvocab.FreightTermsThirdParty,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			FreightTerms(tc.shipment, tc.original)
			require.Equal(t, tc.want, tc.shipment.FreightTerms)
		})
	}

	var none *shipment.Shipment
	FreightTerms(none, nil)
}
