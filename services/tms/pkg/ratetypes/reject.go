package ratetypes

// RejectReason says why a rule that was fetched as a candidate did not end up
// pricing the shipment.
//
// Every reason is recorded on the quote, which is what turns "why did I get
// this rate" from an investigation into a list. The reasons split into two
// families that a reader needs to tell apart: a rule that did not apply at all,
// and a rule that applied but was beaten by a better one.
type RejectReason string

const (
	// RejectReasonNone marks the rule that won.
	RejectReasonNone = RejectReason("")

	RejectReasonLaneMismatch          = RejectReason("LaneMismatch")
	RejectReasonNotEffective          = RejectReason("NotEffective")
	RejectReasonAgreementNotEffective = RejectReason("AgreementNotEffective")
	RejectReasonAgreementInactive     = RejectReason("AgreementInactive")
	RejectReasonRuleInactive          = RejectReason("RuleInactive")
	RejectReasonServiceTypeMismatch   = RejectReason("ServiceTypeMismatch")
	RejectReasonShipmentTypeMismatch  = RejectReason("ShipmentTypeMismatch")
	RejectReasonEquipmentMismatch     = RejectReason("EquipmentMismatch")
	RejectReasonServiceModelMismatch  = RejectReason("ServiceModelMismatch")
	RejectReasonCommodityMismatch     = RejectReason("CommodityMismatch")
	RejectReasonFreightClassMismatch  = RejectReason("FreightClassMismatch")
	RejectReasonWeightOutOfRange      = RejectReason("WeightOutOfRange")
	RejectReasonDistanceOutOfRange    = RejectReason("DistanceOutOfRange")
	RejectReasonStopsOutOfRange       = RejectReason("StopsOutOfRange")
	RejectReasonDayOfWeekExcluded     = RejectReason("DayOfWeekExcluded")
	RejectReasonHazmatRequired        = RejectReason("HazmatRequired")
	RejectReasonTempControlRequired   = RejectReason("TempControlRequired")
	RejectReasonOutsideRadius         = RejectReason("OutsideRadius")

	RejectReasonLostOnPriority      = RejectReason("LostOnPriority")
	RejectReasonLostOnSpecificity   = RejectReason("LostOnSpecificity")
	RejectReasonLostOnEffectiveDate = RejectReason("LostOnEffectiveDate")
	RejectReasonLostOnTiebreak      = RejectReason("LostOnTiebreak")

	RejectReasonPricingError = RejectReason("PricingError")
)

func (rr RejectReason) String() string {
	return string(rr)
}

// Lost reports whether the rule applied to the shipment but was outranked,
// rather than failing to apply at all. The distinction matters to whoever is
// reading a trace: a rule that lost is competition, a rule that did not match
// is a lane the contract does not cover.
func (rr RejectReason) Lost() bool {
	switch rr {
	case RejectReasonLostOnPriority,
		RejectReasonLostOnSpecificity,
		RejectReasonLostOnEffectiveDate,
		RejectReasonLostOnTiebreak:
		return true
	default:
		return false
	}
}

// Explanation is a sentence a person can read without knowing the model.
func (rr RejectReason) Explanation() string {
	switch rr {
	case RejectReasonNone:
		return "Applied"
	case RejectReasonLaneMismatch:
		return "The lane does not cover this origin and destination"
	case RejectReasonNotEffective:
		return "The rate was not in effect on the rating date"
	case RejectReasonAgreementNotEffective:
		return "The agreement was not in effect on the rating date"
	case RejectReasonAgreementInactive:
		return "The agreement is not active"
	case RejectReasonRuleInactive:
		return "The rate is not active"
	case RejectReasonServiceTypeMismatch:
		return "The rate is limited to other service types"
	case RejectReasonShipmentTypeMismatch:
		return "The rate is limited to other shipment types"
	case RejectReasonEquipmentMismatch:
		return "The rate is limited to other equipment"
	case RejectReasonServiceModelMismatch:
		return "The rate is limited to another mode"
	case RejectReasonCommodityMismatch:
		return "The rate is limited to other commodities"
	case RejectReasonFreightClassMismatch:
		return "The rate is limited to other freight classes"
	case RejectReasonWeightOutOfRange:
		return "The shipment weight falls outside the rate's range"
	case RejectReasonDistanceOutOfRange:
		return "The distance falls outside the rate's range"
	case RejectReasonStopsOutOfRange:
		return "The stop count falls outside the rate's range"
	case RejectReasonDayOfWeekExcluded:
		return "The rate does not apply on this day of the week"
	case RejectReasonHazmatRequired:
		return "The rate applies only to hazmat shipments"
	case RejectReasonTempControlRequired:
		return "The rate applies only to temperature controlled shipments"
	case RejectReasonOutsideRadius:
		return "The shipment falls outside the rate's radius"
	case RejectReasonLostOnPriority:
		return "Another rate carried a higher priority"
	case RejectReasonLostOnSpecificity:
		return "Another rate was written more specifically"
	case RejectReasonLostOnEffectiveDate:
		return "Another rate took effect more recently"
	case RejectReasonLostOnTiebreak:
		return "Another rate matched equally well and was created first"
	case RejectReasonPricingError:
		return "The rate could not be calculated"
	default:
		return string(rr)
	}
}
