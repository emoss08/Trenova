package ratezoneservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratezone"
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
	validator *validationframework.TenantedValidator[*ratezone.RateZone]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: newBuilder(p.DB).Build()}
}

func newBuilder(
	db *postgres.Connection,
) *validationframework.TenantedValidatorBuilder[*ratezone.RateZone] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*ratezone.RateZone]().
		WithModelName("Rate Zone")

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return db.DB() }),
		).
		WithUniqueField(
			"code",
			"code",
			"A rate zone with this code already exists in your organization",
			func(z *ratezone.RateZone) any { return z.Code },
		)
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *ratezone.RateZone,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *ratezone.RateZone,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
