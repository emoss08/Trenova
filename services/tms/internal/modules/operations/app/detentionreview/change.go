package detentionreview

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
)

// OccurrenceChange is one occurrence before and after a person's decision,
// computed by the transition the decision itself applies. Shipment is every
// occurrence on the same shipment as it stands, so a preview can say what the
// shipment's detention charge comes to on either side.
type OccurrenceChange struct {
	Before   *detention.DetentionOccurrence
	After    *detention.DetentionOccurrence
	Shipment []*detention.DetentionOccurrence
	Now      int64
}
