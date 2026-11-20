package equipmenttype

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (et *EquipmentType) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	et.Code = format.Code(et.Code)
}
