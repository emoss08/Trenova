package dispatchcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
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
	validator *validationframework.TenantedValidator[*dispatchcontrol.DispatchControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*dispatchcontrol.DispatchControl]().
			WithModelName("DispatchControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
