package ratequote

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

func (q *RateQuote) WithholdFields(visible permission.FieldGroupSet) {
	if q == nil {
		return
	}
	q.Agreement.WithholdFields(visible)
}
