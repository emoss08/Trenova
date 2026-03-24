package shipmentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*tenant.ShipmentControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.ShipmentControl]().
			WithModelName("ShipmentControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			WithOptionalReferenceCheck(
				"detentionChargeId",
				"accessorial_charges",
				"Detention charge does not exist in your organization",
				func(sc *tenant.ShipmentControl) pulid.ID {
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
	entity *tenant.ShipmentControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
