package ratequote

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

// WithholdFields clears the quote's margin and the cost that, beside its
// total, gives the margin away, along with the same figures in its trace.
func (q *RateQuote) WithholdFields(visible permission.FieldGroupSet) {
	if q == nil {
		return
	}
	q.Agreement.WithholdFields(visible)
	if visible.Has(permission.FieldGroupMargins) {
		return
	}
	q.CostAmount = decimal.NullDecimal{}
	q.MarginAmount = decimal.NullDecimal{}
	q.MarginPercent = decimal.NullDecimal{}
	if q.Trace != nil {
		q.Trace.Totals.Cost = decimal.NullDecimal{}
		q.Trace.Totals.Margin = decimal.NullDecimal{}
		q.Trace.Totals.MarginPct = decimal.NullDecimal{}
	}
}
