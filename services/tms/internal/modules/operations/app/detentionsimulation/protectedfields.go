package detentionsimulation

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (r *PreviewResult) WithholdFields(visible permission.FieldGroupSet) {
	if r == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	r.DriverPayAmount = decimal.NullDecimal{}
	r.NetMargin = decimal.NullDecimal{}
	r.Trace.WithholdFields(visible)
}

func (b *BacktestBucket) WithholdFields(visible permission.FieldGroupSet) {
	if b == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	b.DriverPayAmount = decimal.NullDecimal{}
	b.NetMargin = decimal.NullDecimal{}
}

func (r *BacktestResult) WithholdFields(visible permission.FieldGroupSet) {
	if r == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	r.ProposedDriverPay = decimal.NullDecimal{}
	r.ProposedNetMargin = decimal.NullDecimal{}
	r.NegativeMarginStops = nil
	for i := range r.ByCustomer {
		r.ByCustomer[i].WithholdFields(visible)
	}
	for i := range r.ByFacility {
		r.ByFacility[i].WithholdFields(visible)
	}
}
