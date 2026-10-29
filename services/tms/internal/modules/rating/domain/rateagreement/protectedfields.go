package rateagreement

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (ra *RateAgreement) WithholdFields(visible protectedfield.GroupSet) {
	if ra == nil || visible.Has(protectedfield.Margins) {
		return
	}
	ra.MarginFloorPercent = decimal.NullDecimal{}
	ra.MaxPayPercentOfSell = decimal.NullDecimal{}
}

func (ra *RateAgreement) KeepFields(stored *RateAgreement, visible protectedfield.GroupSet) {
	if ra == nil || stored == nil || visible.Has(protectedfield.Margins) {
		return
	}
	ra.MarginFloorPercent = stored.MarginFloorPercent
	ra.MaxPayPercentOfSell = stored.MaxPayPercentOfSell
}

func (v *RateAgreementVersion) WithholdFields(visible protectedfield.GroupSet) {
	if v == nil || visible.Has(protectedfield.Margins) {
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
