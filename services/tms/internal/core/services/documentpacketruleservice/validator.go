package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
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
	validator *validationframework.TenantedValidator[*documentpacketrule.DocumentPacketRule]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*documentpacketrule.DocumentPacketRule]().
			WithModelName("Document Packet Rule").
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
