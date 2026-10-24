package dispatchcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*dispatchcontrol.DispatchControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, dispatchcontroldb.DispatchControlsSchema())

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*dispatchcontrol.DispatchControl]().
			WithModelName("DispatchControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
