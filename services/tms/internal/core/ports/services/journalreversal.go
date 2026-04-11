package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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

type JournalReversalService interface {
	List(ctx context.Context, req *repositories.ListJournalReversalsRequest) (*pagination.ListResult[*journalreversal.Reversal], error)
	Get(ctx context.Context, req *GetJournalReversalRequest) (*journalreversal.Reversal, error)
	Create(ctx context.Context, req *CreateJournalReversalRequest, actor *RequestActor) (*journalreversal.Reversal, error)
	Approve(ctx context.Context, req *GetJournalReversalRequest, actor *RequestActor) (*journalreversal.Reversal, error)
	Reject(ctx context.Context, req *RejectJournalReversalRequest, actor *RequestActor) (*journalreversal.Reversal, error)
	Cancel(ctx context.Context, req *CancelJournalReversalRequest, actor *RequestActor) (*journalreversal.Reversal, error)
	Post(ctx context.Context, req *GetJournalReversalRequest, actor *RequestActor) (*journalreversal.Reversal, error)
}
