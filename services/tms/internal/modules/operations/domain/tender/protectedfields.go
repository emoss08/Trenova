package tender

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (o *TenderOffer) WithholdFields(visible protectedfield.GroupSet) {
	if o == nil || visible.Has(protectedfield.Margins) {
		return
	}
	o.CounterMargin = decimal.NullDecimal{}
}
