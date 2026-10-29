package detention

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

// WithholdFields clears what the driver is paid and the margin it leaves:
// beside the billed amount the pay gives the margin away.
func (o *DetentionOccurrence) WithholdFields(visible protectedfield.GroupSet) {
	if o == nil || visible.Has(protectedfield.Margins) {
		return
	}
	o.DriverPayAmount = decimal.NullDecimal{}
	o.NetMargin = decimal.NullDecimal{}
	o.CalculationTrace.WithholdFields(visible)
}

// WithholdFields drops the driver's pay from the steps that carry it.
func (t *CalculationTrace) WithholdFields(visible protectedfield.GroupSet) {
	if t == nil || visible.Has(protectedfield.Margins) {
		return
	}
	for i := range t.Steps {
		if t.Steps[i].Kind == TraceStepDriverPay {
			t.Steps[i].Amount = nil
		}
	}
}
