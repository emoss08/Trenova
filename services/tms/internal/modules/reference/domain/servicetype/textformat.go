package servicetype

import "github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"

func (st *ServiceType) ApplyTextFormat(format dataentrycontrol.TextFormat) {
	st.Code = format.Code(st.Code)
}
