package rateagreement

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
)

// AccessorialFacts is what a shipment looks like to an accessorial schedule.
//
// It is deliberately small: the tests a contract's own rows can answer without
// help. Anything needing a formula engine stays with the caller.
type AccessorialFacts struct {
	// At is the date the contract's terms are read on.
	At int64

	ServiceTypeID  pulid.ID
	ShipmentTypeID pulid.ID
}

// AutoApplyAccessorials picks the schedule rows that apply to this shipment on
// their own.
//
// The contract decides which of its own rows apply, so the sell side and the
// buy side cannot drift apart on the answer — a rate confirmation and a carrier
// settlement reading two different lists is exactly the complaint contract
// pricing exists to fix.
//
// Rows carrying an ApplyCondition come back rather than being dropped: only the
// caller has a formula engine to answer one with, and answering "no" here would
// silently disable every conditional accessorial in the product. A caller
// without an engine should treat a condition as unmet rather than as absent.
func (ra *RateAgreement) AutoApplyAccessorials(
	facts AccessorialFacts,
) []*RateAgreementAccessorial {
	if ra == nil {
		return nil
	}

	applicable := make([]*RateAgreementAccessorial, 0, len(ra.Accessorials))

	for _, accessorial := range ra.Accessorials {
		if accessorial == nil || !accessorial.AutoApply || accessorial.Waived {
			continue
		}

		if !accessorial.IsEffectiveAt(facts.At) {
			continue
		}

		if !matchesIDSet(accessorial.ServiceTypeIDs, facts.ServiceTypeID) ||
			!matchesIDSet(accessorial.ShipmentTypeIDs, facts.ShipmentTypeID) {
			continue
		}

		applicable = append(applicable, accessorial)
	}

	return applicable
}

// matchesIDSet reports whether a value is allowed by a scoping set.
//
// An empty set means the row does not care, which is the convention every other
// scoped record in the system uses. Reading it as "matches nothing" would make
// every unscoped row silently stop applying.
func matchesIDSet(allowed []pulid.ID, value pulid.ID) bool {
	if len(allowed) == 0 {
		return true
	}

	return slices.Contains(allowed, value)
}
