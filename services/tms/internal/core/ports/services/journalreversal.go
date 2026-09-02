package services

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateJournalReversalRequest struct {
	OriginalJournalEntryID  pulid.ID              `json:"originalJournalEntryId"`
	RequestedAccountingDate int64                 `json:"requestedAccountingDate"`
	ReasonCode              string                `json:"reasonCode"`
	ReasonText              string                `json:"reasonText"`
	TenantInfo              pagination.TenantInfo `json:"tenantInfo"`
}

type GetJournalReversalRequest struct {
	ReversalID pulid.ID              `json:"reversalId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type RejectJournalReversalRequest struct {
	ReversalID pulid.ID              `json:"reversalId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CancelJournalReversalRequest struct {
	ReversalID pulid.ID              `json:"reversalId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}
