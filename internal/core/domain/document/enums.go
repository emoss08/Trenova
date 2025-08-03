/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
