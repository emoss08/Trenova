package fleetcode

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (fc *FleetCode) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	fc.Code = format.Code(fc.Code)
	fc.Description = dataentrycontrol.CleanText(fc.Description)
}
