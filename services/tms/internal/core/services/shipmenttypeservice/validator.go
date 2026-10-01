package shipmenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
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
	validator *validationframework.TenantedValidator[*shipmenttype.ShipmentType]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*shipmenttype.ShipmentType]().
			WithModelName("ShipmentType").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithUniqueField(
				"code",
				"code",
				"Shipment type with this code already exists in your organization",
				func(sht *shipmenttype.ShipmentType) any { return sht.Code },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
