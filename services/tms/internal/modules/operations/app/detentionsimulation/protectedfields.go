package detentionsimulation

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (r *PreviewResult) WithholdFields(visible protectedfield.GroupSet) {
	if r == nil || visible.Has(protectedfield.Margins) {
		return
	}
	r.DriverPayAmount = decimal.NullDecimal{}
	r.NetMargin = decimal.NullDecimal{}
	r.Trace.WithholdFields(visible)
}

func (b *BacktestBucket) WithholdFields(visible protectedfield.GroupSet) {
	if b == nil || visible.Has(protectedfield.Margins) {
		return
	}
	b.DriverPayAmount = decimal.NullDecimal{}
	b.NetMargin = decimal.NullDecimal{}
}

func (r *BacktestResult) WithholdFields(visible protectedfield.GroupSet) {
	if r == nil || visible.Has(protectedfield.Margins) {
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
