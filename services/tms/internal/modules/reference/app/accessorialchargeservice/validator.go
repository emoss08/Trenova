package accessorialchargeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*accessorialcharge.AccessorialCharge]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*accessorialcharge.AccessorialCharge]().
			WithModelName("AccessorialCharge").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
