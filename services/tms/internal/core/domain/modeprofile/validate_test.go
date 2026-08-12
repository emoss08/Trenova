package modeprofile_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validProfile() *modeprofile.Profile {
	return &modeprofile.Profile{
		Name:           "Flatbed Oversize",
		Code:           "FB-OS",
		Status:         modeprofile.ProfileStatusActive,
		ServiceModel:   modeprofile.ServiceModelTruckload,
		EquipmentClass: modeprofile.EquipmentClassFlatbed,
		ExecutionParty: modeprofile.ExecutionPartyCompanyAsset,
	}
}

func TestProfile_AcceptsEveryDeclaredEnumMember(t *testing.T) {
	models := []modeprofile.ServiceModel{
		modeprofile.ServiceModelTruckload,
		modeprofile.ServiceModelDedicated,
		modeprofile.ServiceModelLTL,
		modeprofile.ServiceModelExpedite,
		modeprofile.ServiceModelFinalMile,
		modeprofile.ServiceModelDrayage,
	}

	for _, model := range models {
		t.Run(model.String(), func(t *testing.T) {
			profile := validProfile()
			profile.ServiceModel = model

			multiErr := errortypes.NewMultiError()
			profile.Validate(multiErr)

			assert.Empty(t, validationtest.FieldCodes(multiErr, "serviceModel"))
		})
	}
}

func TestProfile_RejectsAnUnknownEnumMember(t *testing.T) {
	cases := map[string]func(*modeprofile.Profile){
		"status":         func(p *modeprofile.Profile) { p.Status = modeprofile.ProfileStatus("Retired") },
		"serviceModel":   func(p *modeprofile.Profile) { p.ServiceModel = modeprofile.ServiceModel("Barge") },
		"equipmentClass": func(p *modeprofile.Profile) { p.EquipmentClass = modeprofile.EquipmentClass("Hopper") },
		"executionParty": func(p *modeprofile.Profile) { p.ExecutionParty = modeprofile.ExecutionParty("Broker") },
	}

	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			profile := validProfile()
			mutate(profile)

			multiErr := errortypes.NewMultiError()
			profile.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.NotEmpty(t, validationtest.FieldCodes(multiErr, field))
		})
	}
}

// An absent enum is Required's business, not the enum rule's, so a blank value
// must not draw two complaints about the same field.
func TestProfile_ReportsAnAbsentEnumOnceAsRequired(t *testing.T) {
	profile := validProfile()
	profile.ServiceModel = ""

	multiErr := errortypes.NewMultiError()
	profile.Validate(multiErr)

	assert.Len(t, validationtest.FieldCodes(multiErr, "serviceModel"), 1)
}

func TestCapabilityRule_RejectsAnUnknownEnforcementLevel(t *testing.T) {
	rule := &modeprofile.CapabilityRule{
		RuleKey:     modeprofile.RuleKeyTemperatureRange,
		Capability:  modeprofile.CapabilityTemperatureControl,
		Enforcement: tenant.EnforcementLevel("Escalate"),
	}

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "enforcement"))
}

// A capability rule may carry any of the four levels, including Ignore and
// Block, which is what separates it from a deviation.
func TestCapabilityRule_AcceptsEveryEnforcementLevel(t *testing.T) {
	levels := []tenant.EnforcementLevel{
		tenant.EnforcementLevelIgnore,
		tenant.EnforcementLevelWarn,
		tenant.EnforcementLevelRequireReview,
		tenant.EnforcementLevelBlock,
	}

	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			rule := &modeprofile.CapabilityRule{
				RuleKey:     modeprofile.RuleKeyTemperatureRange,
				Capability:  modeprofile.CapabilityTemperatureControl,
				Enforcement: level,
			}

			multiErr := errortypes.NewMultiError()
			rule.Validate(multiErr)

			assert.Empty(t, validationtest.FieldCodes(multiErr, "enforcement"))
		})
	}
}

func validDeviation() *modeprofile.Deviation {
	return &modeprofile.Deviation{
		RuleKey:      modeprofile.RuleKeyTemperatureRange,
		Capability:   modeprofile.CapabilityTemperatureControl,
		Enforcement:  tenant.EnforcementLevelWarn,
		ResourceType: "shipment",
		ResourceID:   "shp_1",
		Message:      "Temperature range missing",
		State:        modeprofile.DeviationStateOpen,
	}
}

func TestDeviation_RejectsAnUnknownState(t *testing.T) {
	deviation := validDeviation()
	deviation.State = modeprofile.DeviationState("Dismissed")

	multiErr := errortypes.NewMultiError()
	deviation.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "state"))
}

// A deviation records an advisory outcome, so only Warn and RequireReview are
// admissible here even though the enum declares four levels. Ignore produced no
// finding to record, and Block stopped the write instead of deviating from it.
// This is deliberately NARROWER than the enum, so it must not be rewritten as a
// plain enum check.
func TestDeviation_AdmitsOnlyAdvisoryEnforcementLevels(t *testing.T) {
	cases := map[tenant.EnforcementLevel]bool{
		tenant.EnforcementLevelWarn:          true,
		tenant.EnforcementLevelRequireReview: true,
		tenant.EnforcementLevelIgnore:        false,
		tenant.EnforcementLevelBlock:         false,
	}

	for level, admissible := range cases {
		t.Run(string(level), func(t *testing.T) {
			deviation := validDeviation()
			deviation.Enforcement = level

			multiErr := errortypes.NewMultiError()
			deviation.Validate(multiErr)

			if admissible {
				assert.Empty(t, validationtest.FieldCodes(multiErr, "enforcement"))
			} else {
				assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "enforcement"),
					"%s must not be recordable as a deviation", level)
			}
		})
	}
}
