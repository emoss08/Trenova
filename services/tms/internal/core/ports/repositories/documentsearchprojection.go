package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentsearchprojection"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentSearchProjectionRepository interface {
	Upsert(ctx context.Context, entity *documentsearchprojection.Projection) (*documentsearchprojection.Projection, error)
	Delete(ctx context.Context, documentID pulid.ID, tenantInfo pagination.TenantInfo) error
}
