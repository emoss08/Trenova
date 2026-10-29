package dispatchcontrol

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (dc *DispatchControl) WithholdFields(visible protectedfield.GroupSet) {
	if dc == nil || visible.Has(protectedfield.Margins) {
		return
	}
	dc.TenderCounterMarginFloorPercent = decimal.NullDecimal{}
}

func (dc *DispatchControl) KeepFields(stored *DispatchControl, visible protectedfield.GroupSet) {
	if dc == nil || stored == nil || visible.Has(protectedfield.Margins) {
		return
	}
	dc.TenderCounterMarginFloorPercent = stored.TenderCounterMarginFloorPercent
}
