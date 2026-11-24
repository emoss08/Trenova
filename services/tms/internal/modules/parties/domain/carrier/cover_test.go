package carrier_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coverPolicy(
	policyType carrier.InsurancePolicyType,
	number string,
	ends int64,
) *carrier.CarrierInsurancePolicy {
	return &carrier.CarrierInsurancePolicy{
		PolicyType:     policyType,
		PolicyNumber:   number,
		ExpirationDate: ends,
	}
}

func TestSoonestRequiredCoverIsTheEarliestOfEachRequiredTypesNewestPolicy(t *testing.T) {
	t.Parallel()

	cover := carrier.SoonestRequiredCover([]*carrier.CarrierInsurancePolicy{
		coverPolicy(carrier.InsurancePolicyTypeAutoLiability, "AL-OLD", 10*day),
		coverPolicy(carrier.InsurancePolicyTypeAutoLiability, "AL-RENEWED", 400*day),
		coverPolicy(carrier.InsurancePolicyTypeCargoLiability, "CL", 30*day),
		coverPolicy(carrier.InsurancePolicyTypeGeneralLiability, "GL", 5*day),
	})

	require.NotNil(t, cover.Policy)
	assert.Equal(t, "CL", cover.Policy.PolicyNumber,
		"a renewal supersedes the old auto policy and general liability is not required")
	assert.Empty(t, cover.Missing)
	assert.True(t, cover.Whole())
}

func TestSoonestRequiredCoverNamesARequiredTypeWithNoPolicy(t *testing.T) {
	t.Parallel()

	cover := carrier.SoonestRequiredCover([]*carrier.CarrierInsurancePolicy{
		coverPolicy(carrier.InsurancePolicyTypeAutoLiability, "AL", 10*day),
	})

	assert.Equal(t, "AL", cover.Policy.PolicyNumber)
	assert.Equal(t,
		[]carrier.InsurancePolicyType{carrier.InsurancePolicyTypeCargoLiability}, cover.Missing)
	assert.False(t, cover.Whole())

	none := carrier.SoonestRequiredCover(nil)
	assert.Nil(t, none.Policy)
	assert.Len(t, none.Missing, len(carrier.RequiredInsurancePolicyTypes))
}
