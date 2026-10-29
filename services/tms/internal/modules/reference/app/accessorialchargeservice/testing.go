package accessorialchargeservice

import (
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*accessorialcharge.AccessorialCharge]().
			WithModelName("AccessorialCharge").
			Build(),
	}
}
