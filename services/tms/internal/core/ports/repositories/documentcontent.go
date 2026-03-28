package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentContentSearchRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceID   string
	ResourceType string
	Query        string
	Limit        int
}

type DocumentContentRepository interface {
	GetByDocumentID(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) (*documentcontent.Content, error)
	ListPagesByDocumentID(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) ([]*documentcontent.Page, error)
	ReplacePages(ctx context.Context, content *documentcontent.Content, pages []*documentcontent.Page) error
	Upsert(ctx context.Context, entity *documentcontent.Content) (*documentcontent.Content, error)
	ListPendingExtraction(ctx context.Context, olderThan int64, limit int) ([]*document.Document, error)
	SearchByResource(ctx context.Context, req *DocumentContentSearchRequest) ([]*document.Document, error)
}
