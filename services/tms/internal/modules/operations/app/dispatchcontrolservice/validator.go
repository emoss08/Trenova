package dispatchcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

type Validator struct {
	validator tenantedvalidation.EntityValidator[*dispatchcontrol.DispatchControl]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*dispatchcontrol.DispatchControl]().
			WithModelName("DispatchControl").
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
