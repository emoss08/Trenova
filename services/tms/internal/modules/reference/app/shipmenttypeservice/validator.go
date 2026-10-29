package shipmenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
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
	validator tenantedvalidation.EntityValidator[*shipmenttype.ShipmentType]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(p.DB, shipmenttypedb.ShipmentTypesSchema())

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*shipmenttype.ShipmentType]().
			WithModelName("ShipmentType").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
