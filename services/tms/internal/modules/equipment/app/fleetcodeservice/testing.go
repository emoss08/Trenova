package fleetcodeservice

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			Build(),
	}
}
