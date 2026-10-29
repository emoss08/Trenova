package fleetcodeservice

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*fleetcode.FleetCode]().
			WithModelName("FleetCode").
			Build(),
	}
}
