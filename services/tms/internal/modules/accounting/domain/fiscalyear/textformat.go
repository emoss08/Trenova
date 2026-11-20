package fiscalyear

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (fy *FiscalYear) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	fy.Name = format.Name(fy.Name)
}
