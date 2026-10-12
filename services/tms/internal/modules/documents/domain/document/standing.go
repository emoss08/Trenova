package document

type Standing string

const (
	StandingAccepted      Standing = "accepted"
	StandingRejected      Standing = "rejected"
	StandingExpired       Standing = "expired"
	StandingPendingReview Standing = "pending_review"
	StandingInactive      Standing = "inactive"
)

func (d *Document) StandingAt(now int64) Standing {
	switch d.Status {
	case StatusRejected:
		return StandingRejected
	case StatusExpired:
		return StandingExpired
	case StatusDraft, StatusPending, StatusPendingApproval:
		return StandingPendingReview
	case StatusArchived:
		return StandingInactive
	case StatusActive:
	}

	if d.ExpirationDate != nil && *d.ExpirationDate <= now {
		return StandingExpired
	}

	return StandingAccepted
}
