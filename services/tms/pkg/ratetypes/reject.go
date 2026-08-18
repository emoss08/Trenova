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
	switch rr { //nolint:exhaustive // every other reason means the rate applied
	case RejectReasonLostOnPriority,
		RejectReasonLostOnSpecificity,
		RejectReasonLostOnEffectiveDate,
		RejectReasonLostOnTiebreak:
		return true
	default:
		return false
	}
}

// rejectExplanations is the sentence shown for each reason.
//
// A map rather than a switch: the entries are pure data, and keeping them in
// one literal means a new reason without a sentence is a visible hole rather
// than a case somebody forgot to add.
var rejectExplanations = map[RejectReason]string{
	RejectReasonNone:                  "Applied",
	RejectReasonLaneMismatch:          "The lane does not cover this origin and destination",
	RejectReasonNotEffective:          "The rate was not in effect on the rating date",
	RejectReasonAgreementNotEffective: "The agreement was not in effect on the rating date",
	RejectReasonAgreementInactive:     "The agreement is not active",
	RejectReasonRuleInactive:          "The rate is not active",
	RejectReasonServiceTypeMismatch:   "The rate is limited to other service types",
	RejectReasonShipmentTypeMismatch:  "The rate is limited to other shipment types",
	RejectReasonEquipmentMismatch:     "The rate is limited to other equipment",
	RejectReasonServiceModelMismatch:  "The rate is limited to another mode",
	RejectReasonCommodityMismatch:     "The rate is limited to other commodities",
	RejectReasonFreightClassMismatch:  "The rate is limited to other freight classes",
	RejectReasonWeightOutOfRange:      "The shipment weight falls outside the rate's range",
	RejectReasonDistanceOutOfRange:    "The distance falls outside the rate's range",
	RejectReasonStopsOutOfRange:       "The stop count falls outside the rate's range",
	RejectReasonDayOfWeekExcluded:     "The rate does not apply on this day of the week",
	RejectReasonHazmatRequired:        "The rate applies only to hazmat shipments",
	RejectReasonTempControlRequired:   "The rate applies only to temperature controlled shipments",
	RejectReasonOutsideRadius:         "The shipment falls outside the rate's radius",
	RejectReasonLostOnPriority:        "Another rate carried a higher priority",
	RejectReasonLostOnSpecificity:     "Another rate was written more specifically",
	RejectReasonLostOnEffectiveDate:   "Another rate took effect more recently",
	RejectReasonLostOnTiebreak:        "Another rate matched equally well and was created first",
	RejectReasonPricingError:          "The rate could not be calculated",
}

// Explanation is a sentence a person can read without knowing the model.
func (rr RejectReason) Explanation() string {
	if explanation, ok := rejectExplanations[rr]; ok {
		return explanation
	}

	return string(rr)
}
