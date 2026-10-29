package shipmentvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	rateCoverageRuleKey       = shipmentvocab.RateCoverageRuleKey
	rateCoverageField         = "formulaTemplateId"
	rateCoverageAdvisoryField = "freightChargeAmount"
)

type unrated int

const (
	priced unrated = iota
	unpriceable
	noRateNoMethod
	noRateWithMethod
)

// createRateCoverageRule reports whether the shipment ended up with a rate, and
// says why when it did not.
//
// The requirement used to live in shipment.Validate as "a formula template is
// required", which stopped being true the moment a contract could price a lane
// on its own. The real rule is that a shipment must have *some* way to be
// priced, and by the time validation runs the rate engine has already tried:
// the shipment carries the outcome, so the rule reads it rather than guessing
// at it.
func createRateCoverageRule(
	coverage RateCoverage,
) validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.
		NewTenantedRule[*shipment.Shipment](rateCoverageRuleKey).
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			ctx context.Context,
			entity *shipment.Shipment,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if verdict := unratedVerdict(entity); verdict != priced && valCtx.IsCreate() &&
				!coverage.acceptsUnrated(ctx, valCtx.OrganizationID) {
				refuseUnrated(multiErr, verdict)
				return nil
			}

			advise(multiErr, rateCoverageAdvisory(entity))
			return nil
		})
}

type RateCoverage struct {
	Billing tenantports.BillingControlRepository
}

func (c RateCoverage) acceptsUnrated(ctx context.Context, orgID pulid.ID) bool {
	if c.Billing == nil {
		return false
	}

	control, err := c.Billing.GetByOrgID(ctx, orgID)
	if err != nil || control == nil {
		return false
	}

	return control.UnratedShipmentDisposition == billingcontrol.UnratedShipmentDispositionZeroAndFlag
}

func (c RateCoverage) refuseUnpricing(
	ctx context.Context,
	original *shipment.Shipment,
	entity *shipment.Shipment,
) *errortypes.MultiError {
	if original == nil || entity == nil || unratedVerdict(original) != priced {
		return nil
	}

	verdict := unratedVerdict(entity)
	if verdict == priced || c.acceptsUnrated(ctx, entity.OrganizationID) {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	refuseUnrated(multiErr, verdict)

	return multiErr
}

func (c RateCoverage) RefuseUnratedCopies(
	ctx context.Context,
	plan *shipmentports.ShipmentDuplicatePlan,
) error {
	if plan == nil || plan.Source == nil {
		return nil
	}

	for _, copied := range plan.Copies {
		if unratedCopyVerdict(plan.Source, copied) == priced {
			continue
		}
		if c.acceptsUnrated(ctx, plan.Source.OrganizationID) {
			return nil
		}

		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Shipment {0} has no rate agreement covering its lane and no rating method, "+
				"so a copy would be priced at zero. Choose a rating method on it first",
			plan.Source.ProNumber,
		)
	}

	return nil
}

func advise(multiErr *errortypes.MultiError, advisory *errortypes.AdvisoryError) {
	if advisory != nil {
		multiErr.AddAdvisory(advisory)
	}
}

func refuseUnrated(multiErr *errortypes.MultiError, verdict unrated) {
	switch verdict {
	case unpriceable:
		multiErr.Add(rateCoverageField, errortypes.ErrRequired,
			"Choose a rating method: this shipment has no rate agreement and no rating method, "+
				"so it cannot be priced")
	case noRateNoMethod, noRateWithMethod:
		multiErr.Add(rateCoverageField, errortypes.ErrRequired,
			"Choose a rating method: no rate agreement covers this lane, "+
				"so this shipment would be priced at zero")
	case priced:
	}
}

// rateCoverageAdvisory names the one thing wrong with how this shipment was
// priced, or nothing when it was priced fine.
func rateCoverageAdvisory(entity *shipment.Shipment) *errortypes.AdvisoryError {
	switch unratedVerdict(entity) {
	case unpriceable:
		return unratedAdvisory(
			"This shipment has no rate agreement and no rating method, so it cannot be priced",
		)
	case noRateNoMethod:
		return unratedAdvisory("No rate agreement covers this lane and no rating method is set, " +
			"so this shipment is priced at zero")
	case noRateWithMethod:
		return unratedAdvisory(
			"No rate agreement covers this lane, so this shipment is priced at zero",
		)
	case priced:
	}

	if entity.RatingDetail != nil &&
		ratequote.Outcome(entity.RatingDetail.Source) == ratequote.OutcomeError {
		return errortypes.NewAdvisory(
			rateCoverageAdvisoryField,
			errortypes.ErrInvalidOperation,
			"The rate could not be calculated: {0}",
			errortypes.SeverityRequireReview, entity.RatingDetail.Explanation,
		).WithRuleKey(rateCoverageRuleKey)
	}

	return nil
}

// unratedVerdict says whether this shipment is priced at nothing, and why. A
// shipment that was never run through the engine — a partial payload, a copy,
// a validation call outside the mutation path — is judged on whether it could
// be priced at all rather than on an outcome it does not have.
func unratedVerdict(entity *shipment.Shipment) unrated {
	if entity.FormulaTemplateID.IsNil() && entity.RateAgreementID == nil {
		switch {
		case entity.RateLocked || pricedByHand(entity):
			return priced
		case entity.RatingDetail == nil:
			return unpriceable
		default:
			return noRateNoMethod
		}
	}

	if entity.RatingDetail == nil ||
		ratequote.Outcome(entity.RatingDetail.Source) != ratequote.OutcomeNoRateFound {
		return priced
	}
	if entity.FormulaTemplateID.IsNil() {
		return noRateNoMethod
	}

	return noRateWithMethod
}

func pricedByHand(entity *shipment.Shipment) bool {
	return entity.FreightChargeAmount.Valid && entity.FreightChargeAmount.Decimal.IsPositive()
}

func unratedCopyVerdict(source, copied *shipment.Shipment) unrated {
	if source == nil || copied == nil {
		return priced
	}

	return unratedVerdict(&shipment.Shipment{
		FormulaTemplateID:   copied.FormulaTemplateID,
		RateAgreementID:     source.RateAgreementID,
		RatingDetail:        source.RatingDetail,
		RateLocked:          source.RateLocked,
		FreightChargeAmount: source.FreightChargeAmount,
	})
}

func unratedAdvisory(message string) *errortypes.AdvisoryError {
	return errortypes.NewAdvisory(
		rateCoverageAdvisoryField,
		errortypes.ErrInvalidOperation,
		message,
		errortypes.SeverityRequireReview,
	).WithRuleKey(rateCoverageRuleKey)
}
