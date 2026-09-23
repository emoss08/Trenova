package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedBillToFromAgreement(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	thirdParty := pulid.MustNew("cus_")
	manual := pulid.MustNew("cus_")
	agreementID := pulid.MustNew("ragr_")
	otherAgreement := pulid.MustNew("ragr_")

	tests := []struct {
		name   string
		entity *shipment.Shipment
		rated  *services.RatedShipment
		want   *pulid.ID
	}{
		{
			name:   "seeds when the agreement is newly applied and no bill-to is set",
			entity: &shipment.Shipment{CustomerID: customerID},
			rated: &services.RatedShipment{
				AgreementID:      &agreementID,
				BillToCustomerID: &thirdParty,
			},
			want: &thirdParty,
		},
		{
			name:   "seeds when a different agreement priced the shipment before",
			entity: &shipment.Shipment{CustomerID: customerID, RateAgreementID: &otherAgreement},
			rated: &services.RatedShipment{
				AgreementID:      &agreementID,
				BillToCustomerID: &thirdParty,
			},
			want: &thirdParty,
		},
		{
			name:   "never overwrites a bill-to somebody set",
			entity: &shipment.Shipment{CustomerID: customerID, BillToCustomerID: &manual},
			rated: &services.RatedShipment{
				AgreementID:      &agreementID,
				BillToCustomerID: &thirdParty,
			},
			want: &manual,
		},
		{
			name:   "does not re-seed on a recalculation under the same agreement",
			entity: &shipment.Shipment{CustomerID: customerID, RateAgreementID: &agreementID},
			rated: &services.RatedShipment{
				AgreementID:      &agreementID,
				BillToCustomerID: &thirdParty,
			},
			want: nil,
		},
		{
			name:   "ignores an agreement that bills the customer themselves",
			entity: &shipment.Shipment{CustomerID: customerID},
			rated: &services.RatedShipment{
				AgreementID:      &agreementID,
				BillToCustomerID: &customerID,
			},
			want: nil,
		},
		{
			name:   "ignores an agreement with no bill-to",
			entity: &shipment.Shipment{CustomerID: customerID},
			rated:  &services.RatedShipment{AgreementID: &agreementID},
			want:   nil,
		},
		{
			name:   "ignores a rating that named no agreement",
			entity: &shipment.Shipment{CustomerID: customerID},
			rated:  &services.RatedShipment{BillToCustomerID: &thirdParty},
			want:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			seedBillToFromAgreement(tc.entity, tc.rated)
			if tc.want == nil {
				assert.Nil(t, tc.entity.BillToCustomerID)
				return
			}
			require.NotNil(t, tc.entity.BillToCustomerID)
			assert.Equal(t, *tc.want, *tc.entity.BillToCustomerID)
		})
	}

	seedBillToFromAgreement(nil, nil)
	seedBillToFromAgreement(&shipment.Shipment{}, nil)
}

func TestApplyQuoteSeedsTheAgreementBillTo(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.CustomerID = pulid.MustNew("cus_")
	thirdParty := pulid.MustNew("cus_")
	agreementID := pulid.MustNew("ragr_")

	(&Calculator{}).applyQuote(entity, &services.RatedShipment{
		AgreementID:      &agreementID,
		BillToCustomerID: &thirdParty,
	})

	require.NotNil(t, entity.BillToCustomerID)
	assert.Equal(t, thirdParty, *entity.BillToCustomerID)
	assert.Equal(t, agreementID, *entity.RateAgreementID)
	assert.Equal(t, thirdParty, entity.PayerID())
}
