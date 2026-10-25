package accessorialchargeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*accessorialcharge.AccessorialCharge]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(
		p.DB,
		accessorialchargedb.AccessorialChargesSchema(),
	)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*accessorialcharge.AccessorialCharge]().
			WithModelName("AccessorialCharge").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithUniqueField(
				"code",
				"code",
				"Accessorial charge with this code already exists in your organization",
				func(acc *accessorialcharge.AccessorialCharge) any { return acc.Code },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
