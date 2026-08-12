package routingguideservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
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
	validator *validationframework.TenantedValidator[*tender.RoutingGuide]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tender.RoutingGuide]().
			WithModelName("RoutingGuide").
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tender.RoutingGuide,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tender.RoutingGuide,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
