package carrierintelsync_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsync"
	"github.com/emoss08/trenova/internal/modules/parties/partiestest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanCarrierSync(t *testing.T) {
	t.Parallel()

	policyID := pulid.MustNew("carins_")
	entity := &carrier.Carrier{
		Name:         "Acme Freight",
		SafetyRating: carrier.SafetyRatingNotRated,
		MCNumber:     "",
		Phone:        "",
		InsurancePolicies: []*carrier.CarrierInsurancePolicy{
			{
				ID:             policyID,
				PolicyType:     carrier.InsurancePolicyTypeAutoLiability,
				PolicyNumber:   "POL-123",
				EffectiveDate:  partiestest.CarrierIntelNow - 100*timeutils.SecondsPerDay,
				ExpirationDate: partiestest.CarrierIntelNow + 200*timeutils.SecondsPerDay,
			},
		},
	}

	profile := partiestest.HealthyCarrierProfile()
	profile.Contacts = &carrierintel.Contacts{Phone: "(555) 123-4567"}
	cancelAt := partiestest.CarrierIntelNow + 10*timeutils.SecondsPerDay
	profile.Insurance.Filings = []carrierintel.InsuranceFiling{
		{
			Type:              carrierintel.InsuranceFilingTypeBIPD,
			PolicyNumber:      "pol-123",
			CancelEffectiveAt: &cancelAt,
		},
		{
			Type:         carrierintel.InsuranceFilingTypeCargo,
			PolicyNumber: "CARGO-9",
			InsurerName:  "Great West",
			Coverage:     partiestest.Decimal(100_000),
		},
	}

	plan := carrierintelsync.PlanCarrierSync(entity, profile, carrierintelsync.SyncSettings{
		AutoApplySafetyRating: true,
		AutoFillFields:        []carrierintelsync.SyncField{carrierintelsync.SyncFieldPhone},
	}, partiestest.CarrierIntelNow)

	autoFields := map[carrierintelsync.SyncField]string{}
	for _, u := range plan.AutoApply {
		autoFields[u.Field] = u.Proposed
	}
	assert.Equal(t, "Satisfactory", autoFields[carrierintelsync.SyncFieldSafetyRating])
	assert.Equal(t, "277621", autoFields[carrierintelsync.SyncFieldMCNumber])
	assert.Equal(t, "5551234567", autoFields[carrierintelsync.SyncFieldPhone])

	name, ok := plan.Suggestion(carrierintelsync.SyncFieldName)
	require.True(t, ok)
	assert.Equal(t, "ACME FREIGHT LLC", name.Proposed)

	require.Len(t, plan.InsuranceAutoApply, 1)
	assert.Equal(
		t,
		carrierintelsync.InsuranceChangeShortenExpiration,
		plan.InsuranceAutoApply[0].Kind,
	)
	assert.Equal(t, cancelAt, *plan.InsuranceAutoApply[0].ProposedExpirationDate)

	require.Len(t, plan.InsuranceSuggestions, 1)
	assert.Equal(t, carrierintelsync.InsuranceChangeNewFiling, plan.InsuranceSuggestions[0].Kind)
	assert.Equal(
		t,
		carrier.InsurancePolicyTypeCargoLiability,
		plan.InsuranceSuggestions[0].PolicyType,
	)

	applied := carrierintelsync.ApplyFieldUpdates(entity, plan.AutoApply)
	assert.Len(t, applied, 3)
	assert.Equal(t, carrier.SafetyRatingSatisfactory, entity.SafetyRating)
	assert.Equal(t, 1, carrierintelsync.ApplyInsuranceChanges(entity, plan.InsuranceAutoApply))
	assert.Equal(t, cancelAt, entity.InsurancePolicies[0].ExpirationDate)
}

func TestPlanCarrierSync_NeverOverwritesWithoutApproval(t *testing.T) {
	t.Parallel()

	entity := &carrier.Carrier{
		Name:         "ACME FREIGHT LLC",
		MCNumber:     "999999",
		Phone:        "5550000000",
		SafetyRating: carrier.SafetyRatingSatisfactory,
	}
	profile := partiestest.HealthyCarrierProfile()
	profile.Contacts = &carrierintel.Contacts{Phone: "5551112222"}
	profile.Safety.Rating = carrierintel.SafetyRatingConditional

	plan := carrierintelsync.PlanCarrierSync(entity, profile, carrierintelsync.SyncSettings{
		AutoFillFields: []carrierintelsync.SyncField{carrierintelsync.SyncFieldPhone},
	}, partiestest.CarrierIntelNow)

	assert.Empty(t, plan.AutoApply)
	_, hasRating := plan.Suggestion(carrierintelsync.SyncFieldSafetyRating)
	_, hasMC := plan.Suggestion(carrierintelsync.SyncFieldMCNumber)
	_, hasPhone := plan.Suggestion(carrierintelsync.SyncFieldPhone)
	_, hasName := plan.Suggestion(carrierintelsync.SyncFieldName)
	assert.True(t, hasRating)
	assert.True(t, hasMC)
	assert.True(t, hasPhone)
	assert.False(t, hasName)
}
