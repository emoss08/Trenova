package tender

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (o *TenderOffer) WithholdFields(visible permission.FieldGroupSet) {
	if o == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	o.CounterMargin = decimal.NullDecimal{}
}
