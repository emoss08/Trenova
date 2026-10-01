package resolver

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The resolved policy crosses to the client as a JSON scalar, so these `json`
// tags are the whole contract — nothing in the GraphQL schema constrains the
// shape. The client validates the payload against resolvedModeProfileSchema in
// client/packages/shared/src/types/shipment.ts.
//
// These assertions are the tripwire for drift. Renaming a field here without
// renaming it there produces a parse failure at runtime and a form that
// silently falls back to "no profile", which is exactly the failure this
// endpoint was added to end.
func TestResolvedPolicyJSON_MatchesTheClientContract(t *testing.T) {
	policy := &modeprofile.ResolvedPolicy{
		ProfileID:      pulid.MustNew("mpf_"),
		ProfileCode:    "FLATBED",
		ProfileName:    "Flatbed",
		ServiceModel:   modeprofile.ServiceModelTruckload,
		EquipmentClass: modeprofile.EquipmentClassFlatbed,
		ExecutionParty: modeprofile.ExecutionPartyCompanyAsset,
		Capabilities: []modeprofile.Capability{
			modeprofile.CapabilityCore,
			modeprofile.CapabilityDimensionalCargo,
		},
		Rules: map[modeprofile.RuleKey]modeprofile.ResolvedRule{
			modeprofile.RuleKeyDimensionsRequired: {
				Key:            modeprofile.RuleKeyDimensionsRequired,
				Capability:     modeprofile.CapabilityDimensionalCargo,
				Label:          "Cargo Dimensions",
				Enforcement:    tenant.EnforcementLevelBlock,
				Enabled:        true,
				Fields:         []string{"commodities"},
				RequiredFields: []string{"commodities"},
				Provenance: modeprofile.Provenance{
					ProfileID:          pulid.MustNew("mpf_"),
					ProfileCode:        "FLATBED",
					ProfileName:        "Flatbed",
					RuleKey:            modeprofile.RuleKeyDimensionsRequired,
					Capability:         modeprofile.CapabilityDimensionalCargo,
					CapabilityLabel:    "Dimensional Cargo",
					Enforcement:        tenant.EnforcementLevelBlock,
					DefaultEnforcement: tenant.EnforcementLevelBlock,
					Rationale:          "Open deck freight cannot be planned without dimensions.",
				},
			},
		},
		ResolvedAt: 1700000000,
	}

	rendered, err := jsonutils.ToJSON(policy)
	require.NoError(t, err)

	for _, key := range []string{
		"profileId", "profileCode", "profileName", "serviceModel",
		"equipmentClass", "executionParty", "capabilities", "rules", "resolvedAt",
	} {
		assert.Contains(t, rendered, key)
	}

	rules, ok := rendered["rules"].(map[string]any)
	require.True(t, ok, "rules must serialise as an object keyed by rule key")

	rule, ok := rules[modeprofile.RuleKeyDimensionsRequired.String()].(map[string]any)
	require.True(t, ok, "rules must be keyed by the rule key string the client looks up")

	for _, key := range []string{
		"key", "capability", "label", "enforcement", "enabled",
		"fields", "requiredFields", "provenance",
	} {
		assert.Contains(t, rule, key)
	}

	// requiredFields has to survive as its own key rather than collapsing into
	// fields; isFieldRequired reads only the former.
	assert.Equal(t, []any{"commodities"}, rule["requiredFields"])
	assert.Equal(t, "Block", rule["enforcement"])

	provenance, ok := rule["provenance"].(map[string]any)
	require.True(t, ok)
	for _, key := range []string{
		"profileId", "profileCode", "profileName", "isOrgDefault", "priority",
		"specificityScore", "ruleKey", "capability", "capabilityLabel",
		"enforcement", "defaultEnforcement", "overridden", "rationale",
	} {
		assert.Contains(t, provenance, key)
	}
}

// A rule that requires nothing must omit requiredFields rather than send an
// empty array or null that the client would have to special-case. The zod
// schema marks it nullish, so absence is the agreed representation.
func TestResolvedPolicyJSON_OmitsEmptyRequiredFields(t *testing.T) {
	rule := modeprofile.ResolvedRule{
		Key:         modeprofile.RuleKeyDuplicateBOL,
		Enforcement: tenant.EnforcementLevelBlock,
		Enabled:     true,
		Fields:      []string{"bol"},
	}

	data, err := sonic.Marshal(rule)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(data, &decoded))

	assert.Contains(t, decoded, "fields")
	assert.NotContains(t, decoded, "requiredFields")
}
