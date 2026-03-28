package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentShipmentDraftRepository interface {
	GetByDocumentID(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) (*documentshipmentdraft.Draft, error)
	Upsert(ctx context.Context, entity *documentshipmentdraft.Draft) (*documentshipmentdraft.Draft, error)
}
