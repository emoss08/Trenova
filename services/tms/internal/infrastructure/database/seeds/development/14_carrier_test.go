package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildSeededCarrier(t *testing.T, def carrierDef, now int64) *carrier.Carrier {
	t.Helper()

	ident := carrierSeedIdentity{
		orgID:        pulid.MustNew("org_"),
		buID:         pulid.MustNew("bu_"),
		carrierID:    pulid.MustNew("car_"),
		stateID:      pulid.MustNew("us_"),
		remitStateID: pulid.MustNew("us_"),
		now:          now,
	}

	entity := buildCarrier(def, ident)
	policies, err := buildCarrierPolicies(def, ident)
	require.NoError(t, err)
	entity.InsurancePolicies = policies
	entity.Contacts = buildCarrierContacts(def, ident)

	return entity
}

func carrierDefByCode(t *testing.T, code string) carrierDef {
	t.Helper()

	for _, def := range carrierDefs() {
		if def.code == code {
			return def
		}
	}

	t.Fatalf("carrier seed does not define code %s", code)
	return carrierDef{}
}

func TestCarrierSeed_EligibilityMatrix(t *testing.T) {
	t.Parallel()

	const now = int64(1_800_000_000)

	tests := []struct {
		code     string
		blocked  bool
		warned   bool
		blockers int
	}{
		{code: "BRT"},
		{code: "SPL", warned: true},
		{code: "IHF", blocked: true, blockers: 1},
		{code: "CSC", blocked: true, blockers: 1},
		{code: "HPT", blocked: true, blockers: 1},
		{code: "LSH", blocked: true, blockers: 1},
		{code: "GPC"},
		{code: "PCT"},
	}

	require.Len(t, carrierDefs(), len(tests))

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			t.Parallel()

			entity := buildSeededCarrier(t, carrierDefByCode(t, tt.code), now)
			result := carrier.EvaluateCarrierEligibility(entity, now)

			assert.Equal(t, tt.blocked, result.IsBlocked(), "blockers: %v", result.Blockers)
			assert.Equal(t, tt.warned, result.HasWarnings(), "warnings: %v", result.Warnings)
			if tt.blocked {
				assert.Len(t, result.Blockers, tt.blockers)
			}
		})
	}
}

func TestCarrierSeed_DefsAreUniqueAndAddressable(t *testing.T) {
	t.Parallel()

	const now = int64(1_800_000_000)

	codes := make(map[string]struct{})
	scacs := make(map[string]struct{})
	dots := make(map[string]struct{})

	for _, def := range carrierDefs() {
		assert.NotContains(t, codes, def.code)
		assert.NotContains(t, scacs, def.scac)
		assert.NotContains(t, dots, def.dotNumber)
		codes[def.code] = struct{}{}
		scacs[def.scac] = struct{}{}
		dots[def.dotNumber] = struct{}{}

		assert.LessOrEqual(t, len(def.code), 10, "carrier %s code exceeds column width", def.name)
		assert.LessOrEqual(t, len(def.scac), 4, "carrier %s SCAC exceeds column width", def.name)
		assert.NotEmpty(t, def.email, "carrier %s needs an email for email-channel offers", def.name)
		assert.NotEmpty(t, def.remitToName, "carrier %s needs remit-to details", def.name)
		require.NotEmpty(t, def.contacts, "carrier %s needs at least one contact", def.name)

		entity := buildSeededCarrier(t, def, now)

		hasRateConRecipient := false
		for _, contact := range entity.Contacts {
			if contact.ReceivesRateConfirmations {
				hasRateConRecipient = true
				assert.NotEmpty(
					t,
					contact.Email,
					"rate confirmation recipient for %s needs an email",
					def.name,
				)
			}
		}
		assert.True(t, hasRateConRecipient, "carrier %s needs a rate confirmation recipient", def.name)

		for _, policy := range entity.InsurancePolicies {
			assert.Greater(
				t,
				policy.ExpirationDate,
				policy.EffectiveDate,
				"policy %s violates the expiration check constraint",
				policy.PolicyNumber,
			)
		}
	}
}
