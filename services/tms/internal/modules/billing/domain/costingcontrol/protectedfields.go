package costingcontrol

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (cc *CostingControl) WithholdFields(visible permission.FieldGroupSet) {
	if cc == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	cc.TargetMarginPercent = decimal.NullDecimal{}
}

func (cc *CostingControl) KeepFields(stored *CostingControl, visible permission.FieldGroupSet) {
	if cc == nil || stored == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	cc.TargetMarginPercent = stored.TargetMarginPercent
}
