package equipmentmanufacturerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*equipmentmanufacturer.EquipmentManufacturer]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*equipmentmanufacturer.EquipmentManufacturer]().
			WithModelName("Equipment Manufacturer").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *equipmentmanufacturer.EquipmentManufacturer,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
