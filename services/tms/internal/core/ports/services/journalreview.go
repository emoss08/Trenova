package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxJournalReviewEntries = 100

type JournalReviewRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	EntryIDs   []pulid.ID            `json:"entryIds"`
}

type JournalReviewOutcome struct {
	EntryID     pulid.ID `json:"entryId"`
	EntryNumber string   `json:"entryNumber"`
	Status      string   `json:"status"`
	Changed     bool     `json:"changed"`
	Error       string   `json:"error"`
}

type JournalReviewResult struct {
	Outcomes []*JournalReviewOutcome `json:"outcomes"`
	Changed  int                     `json:"changed"`
	Failed   int                     `json:"failed"`
}

type ListJournalReviewRequest struct {
	TenantInfo pagination.TenantInfo    `json:"tenantInfo"`
	Filter     *pagination.QueryOptions `json:"filter"`
	Cursor     pagination.CursorInfo    `json:"cursor"`
	Statuses   []journalentry.Status    `json:"statuses"`
}

type JournalReviewSummary struct {
	AwaitingApproval     int                           `json:"awaitingApproval"`
	ReadyToPost          int                           `json:"readyToPost"`
	OldestAccountingDate *int64                        `json:"oldestAccountingDate"`
	PostingMode          tenant.JournalPostingModeType `json:"postingMode"`
	RequiresApproval     bool                          `json:"requiresApproval"`
}

type JournalReviewService interface {
	List(
		ctx context.Context,
		req *ListJournalReviewRequest,
	) (*pagination.CursorListResult[*journalentry.JournalEntry], error)
	Summary(ctx context.Context, tenantInfo pagination.TenantInfo) (*JournalReviewSummary, error)
	Approve(
		ctx context.Context,
		req *JournalReviewRequest,
		actor *RequestActor,
	) (*JournalReviewResult, error)
	Post(
		ctx context.Context,
		req *JournalReviewRequest,
		actor *RequestActor,
	) (*JournalReviewResult, error)
}
