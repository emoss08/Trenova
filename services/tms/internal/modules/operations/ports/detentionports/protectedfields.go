package detentionports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (s *FacilityDetentionStat) WithholdFields(visible protectedfield.GroupSet) {
	if s == nil || visible.Has(protectedfield.Margins) {
		return
	}
	s.DriverPayAmount = decimal.NullDecimal{}
	s.NetMargin = decimal.NullDecimal{}
}

// WithholdFields clears what drivers were paid and the margin it leaves, for a reader without
// the margins group: beside the billed amount the pay gives the margin away.
func (t *IntelligenceTotals) WithholdFields(visible protectedfield.GroupSet) {
	if t == nil || visible.Has(protectedfield.Margins) {
		return
	}
	t.DriverPayAmount = decimal.NullDecimal{}
	t.NetMargin = decimal.NullDecimal{}
}

func (f *FacilityIntelligence) WithholdFields(visible protectedfield.GroupSet) {
	if f == nil || visible.Has(protectedfield.Margins) {
		return
	}
	f.DriverPayAmount = decimal.NullDecimal{}
	f.NetMargin = decimal.NullDecimal{}
}

func (c *CustomerIntelligence) WithholdFields(visible protectedfield.GroupSet) {
	if c == nil || visible.Has(protectedfield.Margins) {
		return
	}
	c.DriverPayAmount = decimal.NullDecimal{}
	c.NetMargin = decimal.NullDecimal{}
}

// WithholdFields clears every margin figure in the rollup.
func (r *IntelligenceRollup) WithholdFields(visible protectedfield.GroupSet) {
	if r == nil || visible.Has(protectedfield.Margins) {
		return
	}
	r.Totals.WithholdFields(visible)
	r.Previous.WithholdFields(visible)
	for i := range r.Facilities {
		r.Facilities[i].WithholdFields(visible)
	}
	for i := range r.Customers {
		r.Customers[i].WithholdFields(visible)
	}
}
