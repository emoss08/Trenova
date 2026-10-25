package core

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
)

func newTemplateConsumers() formulatemplateports.TemplateConsumers {
	return formulatemplateports.TemplateConsumers{Shipments: shipmentdb.Shipments.FormulaTemplateID}
}
