package carrierservice

import (
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*carrier.Carrier]().
			WithModelName("Carrier").
			Build(),
	}
}
