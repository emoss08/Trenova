package shipmentcontrolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.ShipmentControl]().
			WithModelName("ShipmentControl").
			Build(),
	}
}
