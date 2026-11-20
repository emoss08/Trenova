package accounttype

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (a *AccountType) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	a.Code = format.Code(a.Code)
	a.Name = format.Name(a.Name)
}
