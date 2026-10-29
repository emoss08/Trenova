package costingcontrol

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (cc *CostingControl) WithholdFields(visible protectedfield.GroupSet) {
	if cc == nil || visible.Has(protectedfield.Margins) {
		return
	}
	cc.TargetMarginPercent = decimal.NullDecimal{}
}

func (cc *CostingControl) KeepFields(stored *CostingControl, visible protectedfield.GroupSet) {
	if cc == nil || stored == nil || visible.Has(protectedfield.Margins) {
		return
	}
	cc.TargetMarginPercent = stored.TargetMarginPercent
}
