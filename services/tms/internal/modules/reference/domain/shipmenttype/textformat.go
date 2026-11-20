package shipmenttype

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (sht *ShipmentType) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	sht.Code = format.Code(sht.Code)
}
