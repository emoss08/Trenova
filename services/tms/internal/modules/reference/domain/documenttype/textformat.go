package documenttype

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (dt *DocumentType) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	dt.Code = format.Code(dt.Code)
	dt.Name = format.Name(dt.Name)
}
