package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
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
	validator *validationframework.TenantedValidator[*documentpacketrule.DocumentPacketRule]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*documentpacketrule.DocumentPacketRule]().
			WithModelName("Document Packet Rule").
			WithReferenceChecker(validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() })).
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
