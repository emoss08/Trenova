package invoiceadjustment

import "github.com/emoss08/trenova/shared/pulid"

const maxIdempotencyKeyLength = 200

// CanBeRejected says whether an approver may still close the adjustment
// without it taking effect: one waiting for approval, or one whose execution
// failed revalidation and would otherwise sit open forever.
func (s Status) CanBeRejected() bool {
	return s == StatusPendingApproval || s == StatusExecutionFailed
}

// Reject closes the adjustment without effect and frees its idempotency key.
// A rejected adjustment is final, so the same request submitted again (voiding
// the same invoice a second time, say) must create a new adjustment rather
// than hand back the rejected one.
func (a *InvoiceAdjustment) Reject(by pulid.ID, at int64, reason string) {
	a.Status = StatusRejected
	a.ApprovalStatus = ApprovalStatusRejected
	a.RejectedByID = by
	a.RejectedAt = &at
	a.RejectionReason = reason
	a.IdempotencyKey = retiredIdempotencyKey(a.ID, a.IdempotencyKey)
}

func retiredIdempotencyKey(id pulid.ID, key string) string {
	retired := "rejected:" + id.String() + ":" + key
	if len(retired) > maxIdempotencyKeyLength {
		return retired[:maxIdempotencyKeyLength]
	}

	return retired
}
