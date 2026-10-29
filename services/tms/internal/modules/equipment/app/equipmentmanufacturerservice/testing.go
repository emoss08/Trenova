package equipmentmanufacturerservice

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*equipmentmanufacturer.EquipmentManufacturer]().
			WithModelName("Equipment Manufacturer").
			Build(),
	}
}
