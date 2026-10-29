package shipmentcontrolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*shipmentcontrol.ShipmentControl]().
			WithModelName("ShipmentControl").
			Build(),
	}
}
