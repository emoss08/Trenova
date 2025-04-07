package document

// DocumentStatus represents the current status of a document
type DocumentStatus string

const (
	DocumentStatusDraft           = DocumentStatus("Draft")
	DocumentStatusActive          = DocumentStatus("Active")
	DocumentStatusArchived        = DocumentStatus("Archived")
	DocumentStatusExpired         = DocumentStatus("Expired")
	DocumentStatusRejected        = DocumentStatus("Rejected")
	DocumentStatusPendingApproval = DocumentStatus("PendingApproval")
)
