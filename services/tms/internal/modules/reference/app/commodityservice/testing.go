package commodityservice

import (
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*commodity.Commodity]().
			WithModelName("Commodity").
			Build(),
	}
}
