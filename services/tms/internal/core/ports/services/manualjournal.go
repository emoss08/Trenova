package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ManualJournalLineInput struct {
	GLAccountID  pulid.ID `json:"glAccountId"`
	Description  string   `json:"description"`
	DebitAmount  int64    `json:"debitAmount"`
	CreditAmount int64    `json:"creditAmount"`
	CustomerID   pulid.ID `json:"customerId"`
	LocationID   pulid.ID `json:"locationId"`
}

type CreateManualJournalRequest struct {
	Description    string                    `json:"description"`
	Reason         string                    `json:"reason"`
	AccountingDate int64                     `json:"accountingDate"`
	CurrencyCode   string                    `json:"currencyCode"`
	Lines          []*ManualJournalLineInput `json:"lines"`
	TenantInfo     pagination.TenantInfo     `json:"tenantInfo"`
}

type UpdateManualJournalDraftRequest struct {
	RequestID      pulid.ID                  `json:"requestId"`
	Description    string                    `json:"description"`
	Reason         string                    `json:"reason"`
	AccountingDate int64                     `json:"accountingDate"`
	CurrencyCode   string                    `json:"currencyCode"`
	Lines          []*ManualJournalLineInput `json:"lines"`
	TenantInfo     pagination.TenantInfo     `json:"tenantInfo"`
}

type GetManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type RejectManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CancelManualJournalRequest struct {
	RequestID  pulid.ID              `json:"requestId"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ManualJournalService interface {
	List(ctx context.Context, req *repositories.ListManualJournalRequest) (*pagination.ListResult[*manualjournal.Request], error)
	Get(ctx context.Context, req *GetManualJournalRequest) (*manualjournal.Request, error)
	CreateDraft(ctx context.Context, req *CreateManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
	UpdateDraft(ctx context.Context, req *UpdateManualJournalDraftRequest, actor *RequestActor) (*manualjournal.Request, error)
	Submit(ctx context.Context, req *GetManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
	Approve(ctx context.Context, req *GetManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
	Post(ctx context.Context, req *GetManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
	Reject(ctx context.Context, req *RejectManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
	Cancel(ctx context.Context, req *CancelManualJournalRequest, actor *RequestActor) (*manualjournal.Request, error)
}
