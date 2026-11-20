package accessorialcharge

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (a *AccessorialCharge) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	a.Code = format.Code(a.Code)
}
