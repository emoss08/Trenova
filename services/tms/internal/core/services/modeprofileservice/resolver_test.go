package modeprofileservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coreProfile(code string, isDefault bool) *modeprofile.Profile {
	return &modeprofile.Profile{
		ID:             pulid.MustNew("mpf_"),
		Code:           code,
		Name:           code,
		Status:         modeprofile.ProfileStatusActive,
		ServiceModel:   modeprofile.ServiceModelTruckload,
		EquipmentClass: modeprofile.EquipmentClassDryVan,
		ExecutionParty: modeprofile.ExecutionPartyCompanyAsset,
		Capabilities: []modeprofile.Capability{
			modeprofile.CapabilityCore,
			modeprofile.CapabilityTemperatureControl,
		},
		IsOrgDefault: isDefault,
	}
}

func TestResolve_SelectsMostSpecificProfile(t *testing.T) {
	customerID := pulid.MustNew("cus_")

	orgDefault := coreProfile("OTR", true)

	scoped := coreProfile("OTR-ACME", false)
	scoped.CustomerID = &customerID
	scoped.SpecificityScore = modeprofile.SpecificityWeightCustomer

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{scoped, orgDefault},
		&services.ResolveModeProfileRequest{CustomerID: customerID},
		100,
	)

	require.NotNil(t, policy)
	assert.Equal(t, "OTR-ACME", policy.ProfileCode)
	assert.Len(t, policy.Candidates, 2)

	assert.True(t, policy.Candidates[0].Selected)
	assert.Contains(t, policy.Candidates[0].MatchedOn, modeprofile.MatchCustomer)
	assert.Equal(t, rejectionOutranked, policy.Candidates[1].RejectionReason)
}

func TestResolve_RejectsProfileTargetingAnotherCustomer(t *testing.T) {
	otherCustomer := pulid.MustNew("cus_")

	scoped := coreProfile("OTR-OTHER", false)
	scoped.CustomerID = &otherCustomer

	orgDefault := coreProfile("OTR", true)

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{scoped, orgDefault},
		&services.ResolveModeProfileRequest{CustomerID: pulid.MustNew("cus_")},
		100,
	)

	require.NotNil(t, policy)
	assert.Equal(t, "OTR", policy.ProfileCode)
	assert.Equal(t, rejectionCustomer, policy.Candidates[0].RejectionReason)
}

func TestResolve_RejectsProfileOutsideEffectiveWindow(t *testing.T) {
	start := int64(500)

	expired := coreProfile("OTR-FUTURE", true)
	expired.EffectiveStartDate = &start

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{expired},
		&services.ResolveModeProfileRequest{},
		100,
	)

	assert.Nil(t, policy)
}

func TestResolve_UsesCatalogDefaultsForUnconfiguredRules(t *testing.T) {
	profile := coreProfile("OTR", true)

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{profile},
		&services.ResolveModeProfileRequest{},
		100,
	)

	require.NotNil(t, policy)

	rule, ok := policy.Rule(modeprofile.RuleKeyMaxShipmentWeight)
	require.True(t, ok)
	assert.Equal(t, tenant.EnforcementLevelBlock, rule.Enforcement)
	assert.False(t, rule.Provenance.Overridden)
	assert.NotEmpty(t, rule.Provenance.Rationale)

	limit, ok := policy.IntParam(
		modeprofile.RuleKeyMaxShipmentWeight,
		modeprofile.ParamMaxWeight,
	)
	require.True(t, ok)
	assert.Equal(t, int64(modeprofile.DefaultMaxShipmentLimit), limit)
}

func TestResolve_OmitsRulesWhoseCapabilityIsNotDeclared(t *testing.T) {
	profile := coreProfile("OTR", true)
	profile.Capabilities = []modeprofile.Capability{modeprofile.CapabilityCore}

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{profile},
		&services.ResolveModeProfileRequest{},
		100,
	)

	require.NotNil(t, policy)

	_, ok := policy.Rule(modeprofile.RuleKeyTemperatureRange)
	assert.False(t, ok)

	_, ok = policy.Rule(modeprofile.RuleKeyHazmatSegregation)
	assert.False(t, ok)
}

func TestResolve_ConfiguredRuleOverridesCarryProvenance(t *testing.T) {
	profile := coreProfile("OTR", true)
	profile.Rules = []*modeprofile.CapabilityRule{
		{
			RuleKey:        modeprofile.RuleKeyMaxShipmentWeight,
			Capability:     modeprofile.CapabilityCore,
			Enforcement:    tenant.EnforcementLevelWarn,
			Enabled:        true,
			Parameters:     map[string]any{modeprofile.ParamMaxWeight: 95000},
			OverrideReason: "We hold overweight permits for the Gulf Coast lanes.",
		},
	}

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{profile},
		&services.ResolveModeProfileRequest{},
		100,
	)

	require.NotNil(t, policy)

	rule, ok := policy.Rule(modeprofile.RuleKeyMaxShipmentWeight)
	require.True(t, ok)
	assert.Equal(t, tenant.EnforcementLevelWarn, rule.Enforcement)
	assert.True(t, rule.Records())
	assert.False(t, rule.Blocks())

	assert.True(t, rule.Provenance.Overridden)
	assert.Equal(t, tenant.EnforcementLevelBlock, rule.Provenance.DefaultEnforcement)
	assert.Contains(t, rule.Provenance.OverrideReason, "overweight permits")

	limit, ok := policy.IntParam(
		modeprofile.RuleKeyMaxShipmentWeight,
		modeprofile.ParamMaxWeight,
	)
	require.True(t, ok)
	assert.Equal(t, int64(95000), limit)
}

func TestResolve_IgnoredRuleDoesNotApply(t *testing.T) {
	profile := coreProfile("OTR", true)
	profile.Rules = []*modeprofile.CapabilityRule{
		{
			RuleKey:     modeprofile.RuleKeyMoveRemoval,
			Capability:  modeprofile.CapabilityCore,
			Enforcement: tenant.EnforcementLevelIgnore,
			Enabled:     true,
		},
	}

	svc := &service{}
	policy := svc.resolveFromCandidates(
		[]*modeprofile.Profile{profile},
		&services.ResolveModeProfileRequest{},
		100,
	)

	require.NotNil(t, policy)
	assert.False(t, policy.Applies(modeprofile.RuleKeyMoveRemoval))
	assert.Equal(
		t,
		tenant.EnforcementLevelIgnore,
		policy.EnforcementFor(modeprofile.RuleKeyMoveRemoval),
	)
}

func TestResolvedPolicy_NilReceiverIsSafe(t *testing.T) {
	var policy *modeprofile.ResolvedPolicy

	assert.False(t, policy.Blocks(modeprofile.RuleKeyMaxShipmentWeight))
	assert.False(t, policy.Records(modeprofile.RuleKeyMaxShipmentWeight))
	assert.False(t, policy.Applies(modeprofile.RuleKeyMaxShipmentWeight))
	assert.False(t, policy.HasCapability(modeprofile.CapabilityCore))
	assert.Equal(
		t,
		tenant.EnforcementLevelIgnore,
		policy.EnforcementFor(modeprofile.RuleKeyMaxShipmentWeight),
	)
}
