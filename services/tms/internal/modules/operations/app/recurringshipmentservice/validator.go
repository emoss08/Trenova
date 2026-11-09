package recurringshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*recurringshipment.RecurringShipment]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*recurringshipment.RecurringShipment]().
			WithModelName("RecurringShipment").
			WithIntegrityChecker(
				p.Integrity,
			).
			WithUniqueField(
				"name",
				"name",
				"A recurring shipment with this name already exists in your organization",
				func(e *recurringshipment.RecurringShipment) any { return e.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
