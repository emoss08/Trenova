package accessorialchargeservice

import (
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*accessorialcharge.AccessorialCharge]().
			WithModelName("AccessorialCharge").
			Build(),
	}
}
