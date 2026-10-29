package equipmentmanufacturerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
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
	validator tenantedvalidation.EntityValidator[*equipmentmanufacturer.EquipmentManufacturer]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(
		p.DB,
		equipmentmanufacturerdb.EquipmentManufacturersSchema(),
	)

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*equipmentmanufacturer.EquipmentManufacturer]().
			WithModelName("Equipment Manufacturer").
			WithUniquenessChecker(checker).
			WithUniqueField(
				"name",
				"name",
				"Equipment manufacturer with this name already exists in your organization",
				func(e *equipmentmanufacturer.EquipmentManufacturer) any { return e.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
