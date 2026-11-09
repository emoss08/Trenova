package ratezoneservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*ratezone.RateZone]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{validator: newBuilder(p.Integrity).Build()}
}

func newBuilder(
	integrity validationframework.IntegrityChecker,
) *tenantedvalidation.Builder[*ratezone.RateZone] {
	return tenantedvalidation.NewBuilder[*ratezone.RateZone]().
		WithModelName("Rate Zone").
		WithIntegrityChecker(integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *ratezone.RateZone,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
