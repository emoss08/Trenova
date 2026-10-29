package shipmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*shipmentcontrol.ShipmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, tenantdb.ShipmentControlsSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*shipmentcontrol.ShipmentControl]().
			WithModelName("ShipmentControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithOptionalCustomReferenceCheck(
				"detentionChargeId",
				"Detention charge does not exist in your organization",
				func(sc *shipmentcontrol.ShipmentControl) pulid.ID {
					if sc.DetentionChargeID == nil {
						return ""
					}

					return *sc.DetentionChargeID
				},
				keelcheck.TableReferenceCheck(p.DB, "accessorial_charges"),
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipmentcontrol.ShipmentControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
