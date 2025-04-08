package document

// Status represents the current status of a document
type Status string

const (
	StatusDraft           = Status("Draft")
	StatusActive          = Status("Active")
	StatusArchived        = Status("Archived")
	StatusExpired         = Status("Expired")
	StatusRejected        = Status("Rejected")
	StatusPendingApproval = Status("PendingApproval")
)
