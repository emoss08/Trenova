package shipment

import "github.com/emoss08/trenova/shared/pulid"

// ApplyCancel is the shipment as cancelling it leaves the record: canceled,
// by whom, when, and why.
func (s *Shipment) ApplyCancel(canceledByID pulid.ID, canceledAt int64, reason string) {
	s.Status = StatusCanceled
	s.CanceledByID = canceledByID
	s.CanceledAt = &canceledAt
	s.CancelReason = reason
}
