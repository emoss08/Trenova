package equipmenttypeservice

import (
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*equipmenttype.EquipmentType]().
			WithModelName("Equipment Type").
			Build(),
	}
}
