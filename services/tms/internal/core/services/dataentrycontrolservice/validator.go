package dataentrycontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*dataentrycontrol.DataEntryControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*dataentrycontrol.DataEntryControl]().
			WithModelName("DataEntryControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerScoped(p.DB),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerScoped(p.DB),
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
