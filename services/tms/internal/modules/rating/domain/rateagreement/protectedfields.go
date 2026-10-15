package rateagreement

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (ra *RateAgreement) WithholdFields(visible permission.FieldGroupSet) {
	if ra == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	ra.MarginFloorPercent = decimal.NullDecimal{}
	ra.MaxPayPercentOfSell = decimal.NullDecimal{}
}

func (ra *RateAgreement) KeepFields(stored *RateAgreement, visible permission.FieldGroupSet) {
	if ra == nil || stored == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	ra.MarginFloorPercent = stored.MarginFloorPercent
	ra.MaxPayPercentOfSell = stored.MaxPayPercentOfSell
}

func (v *RateAgreementVersion) WithholdFields(visible permission.FieldGroupSet) {
	if v == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	v.MarginFloorPercent = decimal.NullDecimal{}
	v.MaxPayPercentOfSell = decimal.NullDecimal{}
	delete(v.ChangeSummary, "marginFloorPercent")
	delete(v.ChangeSummary, "maxPayPercentOfSell")
	if v.Agreement != nil {
		v.Agreement.WithholdFields(visible)
	}
}
