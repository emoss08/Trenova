package jurisdictionrule_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validRule() *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		StateID:           pulid.MustNew("us_"),
		Status:            jurisdictionrule.StatusActive,
		VerificationState: jurisdictionrule.VerificationUnverified,
	}
}

func TestJurisdictionRule_RejectsAnUnknownStatus(t *testing.T) {
	rule := validRule()
	rule.Status = jurisdictionrule.Status("Retired")

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "status"))
}

func TestJurisdictionRule_AcceptsEveryDeclaredStatus(t *testing.T) {
	for _, status := range []jurisdictionrule.Status{
		jurisdictionrule.StatusActive,
		jurisdictionrule.StatusInactive,
		jurisdictionrule.StatusDraft,
	} {
		t.Run(status.String(), func(t *testing.T) {
			rule := validRule()
			rule.Status = status

			multiErr := errortypes.NewMultiError()
			rule.Validate(multiErr)

			assert.Empty(t, validationtest.FieldCodes(multiErr, "status"))
		})
	}
}

func TestJurisdictionRule_RejectsAnUnknownVerificationState(t *testing.T) {
	rule := validRule()
	rule.VerificationState = jurisdictionrule.VerificationState("Pending")

	multiErr := errortypes.NewMultiError()
	rule.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "verificationState"))
}

// Seeded reference data ships Unverified on purpose, so that state has to be
// writable rather than treated as a placeholder the validator rejects.
func TestJurisdictionRule_AcceptsEveryDeclaredVerificationState(t *testing.T) {
	for _, state := range []jurisdictionrule.VerificationState{
		jurisdictionrule.VerificationUnverified,
		jurisdictionrule.VerificationVerified,
		jurisdictionrule.VerificationDisputed,
	} {
		t.Run(state.String(), func(t *testing.T) {
			rule := validRule()
			rule.VerificationState = state

			multiErr := errortypes.NewMultiError()
			rule.Validate(multiErr)

			assert.Empty(t, validationtest.FieldCodes(multiErr, "verificationState"))
		})
	}
}

func TestEscortThreshold_RejectsUnknownRoleAndTrigger(t *testing.T) {
	threshold := &jurisdictionrule.EscortThreshold{
		Role:    jurisdictionrule.EscortRole("Pilot"),
		Trigger: jurisdictionrule.TriggerKind("Axles"),
	}

	multiErr := errortypes.NewMultiError()
	threshold.Validate(multiErr)

	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "role"))
	assert.NotEmpty(t, validationtest.FieldCodes(multiErr, "trigger"))
}
