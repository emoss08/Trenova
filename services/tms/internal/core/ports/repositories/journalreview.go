package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type LockJournalEntryRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	EntryID    pulid.ID              `json:"entryId"`
}

type ApproveJournalEntryParams struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	EntryID      pulid.ID              `json:"entryId"`
	BatchID      pulid.ID              `json:"batchId"`
	Version      int64                 `json:"version"`
	ApprovedByID pulid.ID              `json:"approvedById"`
	ApprovedAt   int64                 `json:"approvedAt"`
}

type PostJournalEntryParams struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	EntryID        pulid.ID              `json:"entryId"`
	BatchID        pulid.ID              `json:"batchId"`
	Version        int64                 `json:"version"`
	FiscalYearID   pulid.ID              `json:"fiscalYearId"`
	FiscalPeriodID pulid.ID              `json:"fiscalPeriodId"`
	AccountingDate int64                 `json:"accountingDate"`
	PostedByID     pulid.ID              `json:"postedById"`
	PostedAt       int64                 `json:"postedAt"`
	Lines          []JournalPostingLine  `json:"lines"`
}

type JournalReviewRepository interface {
	LockEntry(ctx context.Context, req LockJournalEntryRequest) (*journalentry.JournalEntry, error)
	ApproveEntry(ctx context.Context, params *ApproveJournalEntryParams) error
	PostEntry(ctx context.Context, params *PostJournalEntryParams) error
	ListConnection(
		ctx context.Context,
		req *ListJournalReviewRequest,
	) (*pagination.CursorListResult[*journalentry.JournalEntry], error)
	Summarize(ctx context.Context, tenantInfo pagination.TenantInfo) (*JournalReviewSummary, error)
}

type ListJournalReviewRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	Cursor   pagination.CursorInfo    `json:"cursor"`
	Statuses []journalentry.Status    `json:"statuses"`
}

type JournalReviewSummary struct {
	AwaitingApproval     int    `json:"awaitingApproval"     bun:"awaiting_approval"`
	ReadyToPost          int    `json:"readyToPost"          bun:"ready_to_post"`
	OldestAccountingDate *int64 `json:"oldestAccountingDate" bun:"oldest_accounting_date"`
}
