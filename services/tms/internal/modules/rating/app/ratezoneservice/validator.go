package ratezoneservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone/ratezonedb"
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
	validator tenantedvalidation.EntityValidator[*ratezone.RateZone]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: newBuilder(p.DB).Build()}
}

func newBuilder(
	db *postgres.Connection,
) *tenantedvalidation.Builder[*ratezone.RateZone] {
	builder := tenantedvalidation.NewBuilder[*ratezone.RateZone]().
		WithModelName("Rate Zone")

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			keelcheck.NewChecker(db, ratezonedb.RateZonesSchema()),
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
