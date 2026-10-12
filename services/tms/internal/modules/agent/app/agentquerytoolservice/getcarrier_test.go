package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCarriers struct {
	carrierports.CarrierRepository

	entity *carrier.Carrier
	last   carrierports.GetCarrierByIDRequest
}

func (s *stubCarriers) GetByID(
	_ context.Context,
	req carrierports.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	s.last = req

	return s.entity, nil
}

func TestGetCarrier_ReadsTheInsuranceAndContactsItsSummaryPromises(t *testing.T) {
	t.Parallel()

	carrierID := pulid.MustNew("car_")
	repo := &stubCarriers{entity: &carrier.Carrier{
		ID:   carrierID,
		Name: "Red River Freight",
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{{
			ID:             pulid.MustNew("carins_"),
			CarrierID:      carrierID,
			PolicyType:     carrier.InsurancePolicyType("Cargo"),
			PolicyNumber:   "CG-1182",
			ProviderName:   "Great West Casualty",
			CoverageAmount: decimal.NewFromInt(100_000),
		}},
		Contacts: []*carrier.CarrierContact{{
			ID:        pulid.MustNew("carcon_"),
			CarrierID: carrierID,
			Name:      "Dispatch desk",
			IsPrimary: true,
		}},
	}}
	params := agentParams(
		map[string]any{"carrierId": carrierID.String()},
		permission.SensitivityRestricted,
	)

	result, err := newGetCarrierTool(repo, &fakePermissions{allowed: true}).Query(t.Context(), params)
	require.NoError(t, err)

	assert.True(t, repo.last.IncludeInsurancePolicies)
	assert.True(t, repo.last.IncludeContacts)
	assert.True(t, repo.last.IncludeState)
	document := encodedDocument(t, result)
	policies, ok := document["insurancePolicies"].([]any)
	require.True(t, ok, "the carrier's insurance policies are in the result")
	require.Len(t, policies, 1)
	assert.Equal(t, "CG-1182", policies[0].(map[string]any)["policyNumber"])
	contacts, ok := document["contacts"].([]any)
	require.True(t, ok, "the carrier's contacts are in the result")
	assert.Len(t, contacts, 1)
}
