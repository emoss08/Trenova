package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*rateagreement.RateAgreement]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: newBuilder(p.DB).Build()}
}

func newBuilder(
	db *postgres.Connection,
) *validationframework.TenantedValidatorBuilder[*rateagreement.RateAgreement] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*rateagreement.RateAgreement]().
		WithModelName("Rate Agreement").
		WithCustomRule(activationReadinessRule())

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return db.DB() }),
		).
		WithUniqueField(
			"code",
			"code",
			"A rate agreement with this code already exists in your organization",
			func(a *rateagreement.RateAgreement) any { return a.Code },
		).
		WithReferenceChecker(
			validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return db.DB() }),
		).
		WithOptionalReferenceCheck(
			"customerId",
			"customers",
			"Customer does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return derefID(a.CustomerID) },
		).
		WithOptionalReferenceCheck(
			"carrierId",
			"carriers",
			"Carrier does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return derefID(a.CarrierID) },
		).
		WithOptionalReferenceCheck(
			"billToCustomerId",
			"customers",
			"Bill-to customer does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return derefID(a.BillToCustomerID) },
		)
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

// ValidateAmendment checks a rule change before it is written.
//
// Amendments are the one write that bypasses the entity validator, because they
// do not carry a whole agreement — so the rules the validator would have run
// have to run here instead.
func (v *Validator) ValidateAmendment(
	agreement *rateagreement.RateAgreement,
	req *repositories.AmendRateAgreementRulesRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if req.EffectiveFrom <= 0 {
		multiErr.Add(
			"effectiveFrom",
			errortypes.ErrRequired,
			"An amendment needs the date its rates take effect",
		)
	}

	if len(req.SupersededIDs) == 0 && len(req.Rules) == 0 {
		multiErr.Add(
			"rules",
			errortypes.ErrRequired,
			"An amendment must add or replace at least one rate",
		)
	}

	if agreement.Status == rateagreement.StatusArchived {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"An archived rate agreement cannot be amended",
		)
	}

	for index, rule := range req.Rules {
		if rule == nil {
			continue
		}

		// The rule inherits the agreement's tenancy and party before it is
		// checked, because its own validation reads both — and the repository
		// will stamp them again on insert regardless.
		rule.RateAgreementID = agreement.ID
		rule.OrganizationID = agreement.OrganizationID
		rule.BusinessUnitID = agreement.BusinessUnitID
		rule.PartyType = agreement.PartyType
		rule.PartyID = agreement.PartyID()

		rule.Validate(multiErr.WithIndex("rules", index))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// activationReadinessRule stops an agreement from being saved in a state it
// could never price anything from.
//
// An active agreement with no rules is the failure that looks like it worked:
// shipments resolve nothing, fall through to whatever the organization's
// unrated disposition says, and nobody notices until the invoices go out.
func activationReadinessRule() validationframework.TenantedRule[*rateagreement.RateAgreement] {
	return validationframework.
		NewTenantedRule[*rateagreement.RateAgreement]("rate_agreement_activation_readiness").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status != rateagreement.StatusActive || len(entity.Rules) > 0 {
				return nil
			}

			multiErr.Add(
				"rules",
				errortypes.ErrInvalidOperation,
				"An active rate agreement needs at least one rate, "+
					"otherwise no shipment can price against it",
			)

			return nil
		})
}

// derefID reads an optional identifier as the zero-or-value the reference
// checker expects, since an unset pointer and a nil id mean the same thing to it.
func derefID(id *pulid.ID) pulid.ID {
	if id == nil {
		return pulid.Nil
	}

	return *id
}
