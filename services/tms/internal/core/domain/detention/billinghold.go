package detention

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// BillingHoldError refuses to bill while any of the occurrences holds billing,
// naming every held occurrence so the screen showing the refusal can link each
// one to the charge a person has to decide. It returns nil when none holds.
func BillingHoldError(occurrences []*DetentionOccurrence) error {
	occurrenceIDs := make([]string, 0, len(occurrences))
	shipmentIDs := make([]string, 0, 1)
	seenShipments := make(map[pulid.ID]struct{}, 1)
	for _, occurrence := range occurrences {
		if occurrence == nil || !occurrence.HoldsBilling() {
			continue
		}
		occurrenceIDs = append(occurrenceIDs, occurrence.ID.String())
		if _, seen := seenShipments[occurrence.ShipmentID]; seen {
			continue
		}
		seenShipments[occurrence.ShipmentID] = struct{}{}
		shipmentIDs = append(shipmentIDs, occurrence.ShipmentID.String())
	}
	if len(occurrenceIDs) == 0 {
		return nil
	}

	var err *errortypes.BusinessError
	if len(shipmentIDs) == 1 {
		err = errortypes.NewBusinessError(
			"{0, plural, one {# detention charge on this shipment still needs approval} other {# detention charges on this shipment still need approval}} before the shipment can be billed. Approve or waive {0, plural, one {it} other {them}} on the detention desk first.",
			len(occurrenceIDs),
		)
	} else {
		err = errortypes.NewBusinessError(
			"{0, plural, one {# detention charge} other {# detention charges}} across {1, plural, one {# shipment} other {# shipments}} still need approval before those shipments can be billed. Approve or waive them on the detention desk first.",
			len(occurrenceIDs),
			len(shipmentIDs),
		)
	}

	return err.
		WithParam("detentionOccurrenceIds", strings.Join(occurrenceIDs, ",")).
		WithParam("shipmentIds", strings.Join(shipmentIDs, ","))
}
