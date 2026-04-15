package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetJournalEntryByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListJournalEntriesRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"`
	FiscalYearID        pulid.ID                 `json:"fiscalYearId"`
	FiscalPeriodID      pulid.ID                 `json:"fiscalPeriodId"`
	ReferenceType       string                   `json:"referenceType"`
	Status              string                   `json:"status"`
	AccountingDateStart int64                    `json:"accountingDateStart"`
	AccountingDateEnd   int64                    `json:"accountingDateEnd"`
}

type MarkJournalEntryReversedRequest struct {
	OriginalEntryID pulid.ID `json:"originalEntryId"`
	ReversalEntryID pulid.ID `json:"reversalEntryId"`
	OrganizationID  pulid.ID `json:"organizationId"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"`
	ReversalDate    int64    `json:"reversalDate"`
	ReversalReason  string   `json:"reversalReason"`
	UpdatedByID     pulid.ID `json:"updatedById"`
}

type JournalEntryRepository interface {
	List(ctx context.Context, req *ListJournalEntriesRequest) (*pagination.ListResult[*journalentry.Entry], error)
	GetByID(ctx context.Context, req GetJournalEntryByIDRequest) (*journalentry.Entry, error)
	MarkReversed(ctx context.Context, req MarkJournalEntryReversedRequest) error
}
