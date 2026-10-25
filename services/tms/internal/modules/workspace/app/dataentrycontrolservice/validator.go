package dataentrycontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol/dataentrycontroldb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*dataentrycontrol.DataEntryControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(
		p.DB,
		dataentrycontroldb.DataEntryControlsSchema(),
	)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*dataentrycontrol.DataEntryControl]().
			WithModelName("DataEntryControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
