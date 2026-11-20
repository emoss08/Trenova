package invoiceadjustmentcontrolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validControl() *invoiceadjustmentcontrol.InvoiceAdjustmentControl {
	return invoiceadjustmentcontrol.NewDefaultInvoiceAdjustmentControl(
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
	)
}

func TestApprovalThresholdRule_KeepsTheRerateToleranceWithinWhatTheColumnHolds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tolerance string
		refused   bool
	}{
		{name: "zero", tolerance: "0"},
		{name: "the largest the column holds", tolerance: "999.999999"},
		{name: "one step past the column", tolerance: "1000", refused: true},
		{name: "far past the column", tolerance: "123456789", refused: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validControl()
			entity.RerateVarianceTolerancePercent = decimal.RequireFromString(tt.tolerance)

			result := validationtest.RunRule(
				t,
				createApprovalThresholdRule(),
				entity,
				validationtest.Update(),
			)

			if !tt.refused {
				result.Exactly()
				return
			}
			result.Exactly(
				validationtest.At("rerateVarianceTolerancePercent", errortypes.ErrInvalid),
			)
			assert.Equal(
				t,
				[]string{"Rerate variance tolerance percent must be at most 999.999999"},
				result.Messages("rerateVarianceTolerancePercent"),
			)
		})
	}
}

func TestApprovalThresholdRule_RefusesANegativeRerateTolerance(t *testing.T) {
	t.Parallel()

	entity := validControl()
	entity.RerateVarianceTolerancePercent = decimal.RequireFromString("-0.000001")

	validationtest.RunRule(t, createApprovalThresholdRule(), entity, validationtest.Update()).
		Exactly(validationtest.At("rerateVarianceTolerancePercent", errortypes.ErrInvalid))
}

func TestClosedPeriodRule_AcceptsEveryEligibilityPolicyTheControlOffers(t *testing.T) {
	t.Parallel()

	for _, policy := range []invoiceadjustmentcontrol.AdjustmentEligibilityPolicy{
		invoiceadjustmentcontrol.AdjustmentEligibilityDisallow,
		invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithApproval,
	} {
		entity := validControl()
		entity.PaidInvoiceAdjustmentPolicy = policy
		entity.PartiallyPaidInvoiceAdjustmentPolicy = policy
		entity.DisputedInvoiceAdjustmentPolicy = policy

		validationtest.RunRule(t, createClosedPeriodRule(), entity, validationtest.Update()).
			Exactly()
	}
}

func TestValidate_RefusesAnEligibilityPolicyTheControlDoesNotKnow(t *testing.T) {
	t.Parallel()

	entity := validControl()
	entity.PaidInvoiceAdjustmentPolicy = invoiceadjustmentcontrol.AdjustmentEligibilityPolicy(
		"AllowWithoutApproval",
	)

	result := validationtest.Validate(t, entity)

	result.Exactly(validationtest.At("paidInvoiceAdjustmentPolicy", errortypes.ErrInvalid))
	assert.Equal(
		t,
		[]string{"Choose one of the listed options"},
		result.Messages("paidInvoiceAdjustmentPolicy"),
	)
}

func TestValidate_AcceptsTheDefaultControl(t *testing.T) {
	t.Parallel()

	validationtest.Validate(t, validControl()).Exactly()
}
