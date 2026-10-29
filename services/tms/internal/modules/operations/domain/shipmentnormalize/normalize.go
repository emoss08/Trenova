package shipmentnormalize

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

// BillTo folds a bill-to that names the ordering customer back to
// "same as customer", so the two spellings never diverge in storage.
func BillTo(s *shipment.Shipment) {
	if s == nil || s.BillToCustomerID == nil {
		return
	}
	if s.BillToCustomerID.IsNil() || *s.BillToCustomerID == s.CustomerID {
		s.BillToCustomerID = nil
	}
}

// FollowCustomerChange keeps "same as customer" meaning the new customer: a
// bill-to that pointed at the old ordering customer was never a third party,
// so it moves with the customer rather than pinning the old one as payer.
func FollowCustomerChange(original, updated *shipment.Shipment) {
	if original == nil || updated == nil || original.CustomerID == updated.CustomerID {
		return
	}
	if updated.BillToCustomerID != nil && *updated.BillToCustomerID == original.CustomerID {
		updated.BillToCustomerID = nil
	}
}

func FreightTerms(s, original *shipment.Shipment) {
	if s == nil || s.FreightTerms != "" {
		return
	}

	if original != nil && original.FreightTerms != "" {
		s.FreightTerms = original.FreightTerms
		return
	}

	s.FreightTerms = shipmentvocab.FreightTermsPrepaid
}

func EntryMethod(s, original *shipment.Shipment) {
	if s == nil || s.EntryMethod != "" {
		return
	}

	if original != nil && original.EntryMethod != "" {
		s.EntryMethod = original.EntryMethod
		return
	}

	s.EntryMethod = shipmentvocab.EntryMethodManual
}
