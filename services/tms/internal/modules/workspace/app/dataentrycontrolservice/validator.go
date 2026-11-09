package dataentrycontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

type Validator struct {
	validator tenantedvalidation.EntityValidator[*dataentrycontrol.DataEntryControl]
}

func NewValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*dataentrycontrol.DataEntryControl]().
			WithModelName("DataEntryControl").
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *dataentrycontrol.DataEntryControl,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
