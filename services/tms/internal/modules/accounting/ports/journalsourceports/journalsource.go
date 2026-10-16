package journalsourceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetJournalSourceByObjectRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	SourceObjectType string                `json:"sourceObjectType"`
	SourceObjectID   string                `json:"sourceObjectId"`
}

type GetJournalSourceByEntryRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	JournalEntryID pulid.ID              `json:"journalEntryId"`
}

type JournalSourceRepository interface {
	GetByObject(
		ctx context.Context,
		req GetJournalSourceByObjectRequest,
	) (*journalsource.Source, error)
	GetByObjectAndEvent(
		ctx context.Context,
		req GetJournalSourceByObjectRequest,
		sourceEventType string,
	) (*journalsource.Source, error)
	ListByObject(
		ctx context.Context,
		req GetJournalSourceByObjectRequest,
	) ([]*journalsource.Source, error)
	GetByJournalEntry(
		ctx context.Context,
		req GetJournalSourceByEntryRequest,
	) (*journalsource.Source, error)
}
