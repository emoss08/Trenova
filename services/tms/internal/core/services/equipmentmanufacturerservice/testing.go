package equipmentmanufacturerservice

import (
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*equipmentmanufacturer.EquipmentManufacturer]().
			WithModelName("Equipment Manufacturer").
			Build(),
	}
}
