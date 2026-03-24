package holdreasonservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*holdreason.HoldReason]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*holdreason.HoldReason]().
			WithModelName("HoldReason").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithUniqueField(
				"code",
				"code",
				"Hold reason with this code already exists in your organization",
				func(e *holdreason.HoldReason) any { return e.Code },
			).
			WithUniqueField(
				"label",
				"label",
				"Hold reason with this label already exists in your organization",
				func(e *holdreason.HoldReason) any { return e.Label },
			).
			WithCustomRule(
				validationframework.
					NewTenantedRule[*holdreason.HoldReason]("severity_validation").
					OnBoth().
					WithValidation(validateBlockingSeverity),
			).
			Build(),
	}
}

func validateBlockingSeverity(
	_ context.Context,
	entity *holdreason.HoldReason,
	_ *validationframework.TenantedValidationContext,
	multiErr *errortypes.MultiError,
) error {
	if entity.DefaultSeverity == holdreason.HoldSeverityBlocking &&
		!entity.DefaultBlocksDispatch && !entity.DefaultBlocksDelivery && !entity.DefaultBlocksBilling {
		multiErr.Add(
			"defaultSeverity",
			errortypes.ErrInvalid,
			"Blocking severity requires at least one gating rule to be enabled (dispatch, delivery, or billing)",
		)
	}
	return nil
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *holdreason.HoldReason,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *holdreason.HoldReason,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
