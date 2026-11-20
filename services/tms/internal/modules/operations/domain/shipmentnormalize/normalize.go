package shipmentnormalize

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/temperaturerange"
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

func LineTemperatures(s *shipment.Shipment) []temperaturerange.Range {
	ranges := make([]temperaturerange.Range, 0)
	for _, move := range s.Moves {
		if move == nil || move.Status == shipmentvocab.MoveStatusCanceled {
			continue
		}
		for _, stop := range move.ActiveStopsBySequence() {
			if line := temperaturerange.Of(stop.TemperatureMin, stop.TemperatureMax); line.Set() {
				ranges = append(ranges, line)
			}
		}
	}
	for _, commodity := range s.Commodities {
		if commodity == nil {
			continue
		}
		line := temperaturerange.Of(commodity.TemperatureMin, commodity.TemperatureMax)
		if line.Set() {
			ranges = append(ranges, line)
		}
	}
	return ranges
}

func Temperature(s *shipment.Shipment) bool {
	if s == nil {
		return false
	}
	lines := LineTemperatures(s)
	if len(lines) == 0 {
		return false
	}
	load := temperaturerange.Load(lines)
	s.TemperatureMin = load.Min
	s.TemperatureMax = load.Max
	return true
}
