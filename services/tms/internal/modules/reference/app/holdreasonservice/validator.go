package holdreasonservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*holdreason.HoldReason]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, holdreasondb.HoldReasonsSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*holdreason.HoldReason]().
			WithModelName("HoldReason").
			WithUniquenessChecker(checker).
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
