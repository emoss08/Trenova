package dispatchcontrol

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (dc *DispatchControl) WithholdFields(visible permission.FieldGroupSet) {
	if dc == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	dc.TenderCounterMarginFloorPercent = decimal.NullDecimal{}
}

func (dc *DispatchControl) KeepFields(stored *DispatchControl, visible permission.FieldGroupSet) {
	if dc == nil || stored == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	dc.TenderCounterMarginFloorPercent = stored.TenderCounterMarginFloorPercent
}
