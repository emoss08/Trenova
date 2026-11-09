package shipmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*shipmentcontrol.ShipmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*shipmentcontrol.ShipmentControl]().
			WithModelName("ShipmentControl").
			WithIntegrityChecker(p.Integrity).
			WithOptionalReferenceCheck(
				"detentionChargeId",
				"accessorial_charges",
				"Detention charge does not exist in your organization",
				func(sc *shipmentcontrol.ShipmentControl) pulid.ID {
					if sc.DetentionChargeID == nil {
						return ""
					}

					return *sc.DetentionChargeID
				},
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipmentcontrol.ShipmentControl,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
