package dataentrycontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol/dataentrycontroldb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*dataentrycontrol.DataEntryControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(
		p.DB,
		dataentrycontroldb.DataEntryControlsSchema(),
	)

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*dataentrycontrol.DataEntryControl]().
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
