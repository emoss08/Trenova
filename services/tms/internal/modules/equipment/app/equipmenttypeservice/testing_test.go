package equipmenttypeservice

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*equipmenttype.EquipmentType]().
			WithModelName("Equipment Type").
			Build(),
	}
}
