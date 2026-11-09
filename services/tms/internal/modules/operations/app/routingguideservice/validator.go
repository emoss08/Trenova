package routingguideservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

type Validator struct {
	validator tenantedvalidation.EntityValidator[*tender.RoutingGuide]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*tender.RoutingGuide]().
			WithModelName("RoutingGuide").
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tender.RoutingGuide,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tender.RoutingGuide,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
