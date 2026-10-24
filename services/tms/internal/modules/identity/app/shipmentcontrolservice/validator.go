package shipmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*tenant.ShipmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(p.DB, tenantdb.ShipmentControlsSchema())

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.ShipmentControl]().
			WithModelName("ShipmentControl").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
			WithOptionalCustomReferenceCheck(
				"detentionChargeId",
				"Detention charge does not exist in your organization",
				func(sc *tenant.ShipmentControl) pulid.ID {
					if sc.DetentionChargeID == nil {
						return ""
					}

					return *sc.DetentionChargeID
				},
				validationframework.KeelTableReferenceCheck(p.DB, "accessorial_charges"),
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.ShipmentControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
