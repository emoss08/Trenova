package customerservice

import (
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*customer.Customer]().
			WithModelName("Customer").
			Build(),
	}
}
