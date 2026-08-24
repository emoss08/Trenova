package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

const rateCoverageRuleKey = "rate_coverage"

// createRateCoverageRule reports whether the shipment ended up with a rate, and
// says why when it did not.
//
// The requirement used to live in shipment.Validate as "a formula template is
// required", which stopped being true the moment a contract could price a lane
// on its own. The real rule is that a shipment must have *some* way to be
// priced, and by the time validation runs the rate engine has already tried:
// the shipment carries the outcome, so the rule reads it rather than guessing
// at it.
//
// It advises rather than blocks. An organization that wants unrated shipments
// refused says so through BillingControl.UnratedShipmentDisposition, which the
// engine enforces before this point; making the same condition fatal twice
// would take that choice away from them.
func createRateCoverageRule() validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment](rateCoverageRuleKey).
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			_ context.Context,
			entity *shipment.Shipment,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			advise(multiErr, rateCoverageAdvisory(entity))
			return nil
		})
}

func advise(multiErr *errortypes.MultiError, advisory *errortypes.AdvisoryError) {
	if advisory != nil {
		multiErr.AddAdvisory(advisory)
	}
}

// rateCoverageAdvisory names the one thing wrong with how this shipment was
// priced, or nothing when it was priced fine.
func rateCoverageAdvisory(entity *shipment.Shipment) *errortypes.AdvisoryError {
	// A shipment that was never run through the engine — a partial payload, a
	// validation call outside the mutation path — is judged on whether it could
	// be priced at all rather than on an outcome it does not have.
	if entity.RatingDetail == nil {
		if entity.FormulaTemplateID.IsNil() && entity.RateAgreementID == nil {
			return unratedAdvisory(
				"This shipment has no rate agreement and no rating method, so it cannot be priced",
			)
		}

		return nil
	}

	outcome := ratequote.Outcome(entity.RatingDetail.Source)

	switch outcome { //nolint:exhaustive // a priced outcome needs no advisory
	case ratequote.OutcomeNoRateFound:
		return unratedAdvisory(noRateFoundMessage(entity))

	case ratequote.OutcomeError:
		return errortypes.NewAdvisory(
			"freightChargeAmount",
			errortypes.ErrInvalidOperation,
			"The rate could not be calculated: "+entity.RatingDetail.Explanation,
			errortypes.SeverityRequireReview,
		).WithRuleKey(rateCoverageRuleKey)

	default:
		return nil
	}
}

func noRateFoundMessage(entity *shipment.Shipment) string {
	if entity.FormulaTemplateID.IsNil() {
		return "No rate agreement covers this lane and no rating method is set, " +
			"so this shipment is priced at zero"
	}

	return "No rate agreement covers this lane, so this shipment is priced at zero"
}

func unratedAdvisory(message string) *errortypes.AdvisoryError {
	return errortypes.NewAdvisory(
		"freightChargeAmount",
		errortypes.ErrInvalidOperation,
		message,
		errortypes.SeverityRequireReview,
	).WithRuleKey(rateCoverageRuleKey)
}
