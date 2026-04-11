package journalreversal

import "github.com/emoss08/trenova/shared/pulid"

type Reversal struct {
	ID                      pulid.ID `json:"id"`
	OrganizationID          pulid.ID `json:"organizationId"`
	BusinessUnitID          pulid.ID `json:"businessUnitId"`
	OriginalJournalEntryID  pulid.ID `json:"originalJournalEntryId"`
	ReversalJournalEntryID  pulid.ID `json:"reversalJournalEntryId"`
	PostedBatchID           pulid.ID `json:"postedBatchId"`
	Status                  Status   `json:"status"`
	RequestedAccountingDate int64    `json:"requestedAccountingDate"`
	ResolvedFiscalYearID    pulid.ID `json:"resolvedFiscalYearId"`
	ResolvedFiscalPeriodID  pulid.ID `json:"resolvedFiscalPeriodId"`
	ReasonCode              string   `json:"reasonCode"`
	ReasonText              string   `json:"reasonText"`
	RequestedByID           pulid.ID `json:"requestedById"`
	ApprovedByID            pulid.ID `json:"approvedById"`
	ApprovedAt              *int64   `json:"approvedAt"`
	RejectedByID            pulid.ID `json:"rejectedById"`
	RejectedAt              *int64   `json:"rejectedAt"`
	RejectionReason         string   `json:"rejectionReason"`
	CancelledByID           pulid.ID `json:"cancelledById"`
	CancelledAt             *int64   `json:"cancelledAt"`
	CancelReason            string   `json:"cancelReason"`
	PostedByID              pulid.ID `json:"postedById"`
	PostedAt                *int64   `json:"postedAt"`
	CreatedAt               int64    `json:"createdAt"`
	UpdatedAt               int64    `json:"updatedAt"`
	Version                 int64    `json:"version"`
}
