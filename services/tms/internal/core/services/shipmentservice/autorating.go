package shipmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

// disengageAutoRating decides whether this save ends the contract's claim on a
// shipment, and reports whether it did.
//
// The contract wrote a rating method, a base rate and its own accessorial rows,
// and from then on those are ordinary fields anybody may edit. Editing one is
// the whole of "overriding the rate" in this system: there is no separate
// override to set, because there is no re-rating for it to defend against. The
// flag comes down here rather than after the recalculation so the rating detail
// that save produces no longer claims a contract priced it.
func disengageAutoRating(original, updated *shipment.Shipment) bool {
	if original == nil || updated == nil || !original.AutoRated {
		return false
	}

	if !shipment.RatedFieldsEdited(original, updated) {
		return false
	}

	updated.ClearAutoRating()

	return true
}
