package detentionports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/shopspring/decimal"
)

func (s *FacilityDetentionStat) WithholdFields(visible protectedfield.GroupSet) {
	if s == nil || visible.Has(protectedfield.Margins) {
		return
	}
	s.DriverPayAmount = decimal.NullDecimal{}
	s.NetMargin = decimal.NullDecimal{}
}

func (s *CustomerDetentionStat) WithholdFields(visible protectedfield.GroupSet) {
	if s == nil || visible.Has(protectedfield.Margins) {
		return
	}
	s.DriverPayAmount = decimal.NullDecimal{}
	s.NetMargin = decimal.NullDecimal{}
}
