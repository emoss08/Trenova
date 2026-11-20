package hazardousmaterial

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (hm *HazardousMaterial) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	hm.Code = format.Code(hm.Code)
	hm.Name = format.Name(hm.Name)
}
