package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentSearchProjectionService interface {
	Upsert(ctx context.Context, doc *document.Document, contentText string) error
	Delete(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) error
}
