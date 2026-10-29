package recurringshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment/recurringshipmentdb"
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
	validator tenantedvalidation.EntityValidator[*recurringshipment.RecurringShipment]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*recurringshipment.RecurringShipment]().
			WithModelName("RecurringShipment").
			WithUniquenessChecker(
				keelcheck.NewChecker(p.DB, recurringshipmentdb.RecurringShipmentsSchema()),
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
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *recurringshipment.RecurringShipment,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
