package commodityservice

import (
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*commodity.Commodity]().
			WithModelName("Commodity").
			Build(),
	}
}
