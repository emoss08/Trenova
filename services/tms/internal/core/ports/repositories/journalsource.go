package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetJournalSourceByObjectRequest struct {
	TenantInfo       pagination.TenantInfo `json:"tenantInfo"`
	SourceObjectType string                `json:"sourceObjectType"`
	SourceObjectID   string                `json:"sourceObjectId"`
}

type JournalSourceRepository interface {
	GetByObject(ctx context.Context, req GetJournalSourceByObjectRequest) (*journalsource.Source, error)
	GetByObjectAndEvent(ctx context.Context, req GetJournalSourceByObjectRequest, sourceEventType string) (*journalsource.Source, error)
}
