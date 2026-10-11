package rateagreement

// Specificity decides which of two rules that both match a shipment gets to
// price it. The weights are stated here, once, and are deliberately boring:
// they are constants, not anything derived from data, because a score that
// moved when somebody edited a zone would let the same shipment rate two ways
// on two days.
//
// Geography outranks everything else. A rule written for a single postal code
// beats a state-wide rule that also happens to name the commodity, because that
// is what a pricing analyst means when they write the narrower lane. It is the
// most argued rule in any rating engine, so both the score and the list of
// things a rule matched on are recorded on every quote — a disagreement should
// be a conversation about the data, not a bug report.
//
// The two ends of a lane are weighted equally. An origin-specific rule and a
// destination-specific rule of the same granularity are equally narrow, and
// there is no principled reason to prefer one; where they do tie, the ordering
// falls through to conditions, then explicit priority, then the effective date,
// and finally the rule id — so the winner is always the same rule.
const (
	// GeographyScale lifts the lane term clear of every condition term. The
	// conditions below sum to 1023, so one step of geographic granularity can
	// never be outvoted by any combination of them.
	GeographyScale = int32(1024)

	ConditionWeightCommodity     = int32(512)
	ConditionWeightFreightClass  = int32(256)
	ConditionWeightEquipmentType = int32(128)
	ConditionWeightServiceType   = int32(64)
	ConditionWeightShipmentType  = int32(32)
	ConditionWeightWeightRange   = int32(16)
	ConditionWeightDistanceRange = int32(8)
	ConditionWeightServiceModel  = int32(4)
	ConditionWeightHazmat        = int32(2)
	ConditionWeightTempControl   = int32(1)

	// MaxConditionScore is every condition weight summed. It is asserted
	// against GeographyScale in the tests so the two can never drift into a
	// state where conditions could outrank geography.
	MaxConditionScore = ConditionWeightCommodity +
		ConditionWeightFreightClass +
		ConditionWeightEquipmentType +
		ConditionWeightServiceType +
		ConditionWeightShipmentType +
		ConditionWeightWeightRange +
		ConditionWeightDistanceRange +
		ConditionWeightServiceModel +
		ConditionWeightHazmat +
		ConditionWeightTempControl
)
