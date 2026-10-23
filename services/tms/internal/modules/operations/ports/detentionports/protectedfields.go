package detentionports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
)

func (s *FacilityDetentionStat) WithholdFields(visible permission.FieldGroupSet) {
	if s == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	s.DriverPayAmount = decimal.NullDecimal{}
	s.NetMargin = decimal.NullDecimal{}
}

func (s *CustomerDetentionStat) WithholdFields(visible permission.FieldGroupSet) {
	if s == nil || visible.Has(permission.FieldGroupMargins) {
		return
	}
	s.DriverPayAmount = decimal.NullDecimal{}
	s.NetMargin = decimal.NullDecimal{}
}
