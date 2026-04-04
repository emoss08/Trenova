package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentContentService interface {
	GetContent(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
	GetShipmentDraft(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentshipmentdraft.DocumentShipmentDraft, error)
	SearchDocuments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		resourceType, resourceID, query string,
	) ([]*document.Document, error)
	Reextract(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) error
	EnqueueExtraction(ctx context.Context, doc *document.Document, userID pulid.ID) error
}
