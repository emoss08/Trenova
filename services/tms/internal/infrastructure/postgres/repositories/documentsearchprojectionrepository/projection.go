package documentsearchprojectionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.DocumentSearchProjectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-search-projection-repository"),
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentsearchprojection.Projection,
) (*documentsearchprojection.Projection, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("id", "organization_id", "business_unit_id") DO UPDATE`).
		Set("resource_id = EXCLUDED.resource_id").
		Set("resource_type = EXCLUDED.resource_type").
		Set("file_name = EXCLUDED.file_name").
		Set("original_name = EXCLUDED.original_name").
		Set("description = EXCLUDED.description").
		Set("tags = EXCLUDED.tags").
		Set("status = EXCLUDED.status").
		Set("content_status = EXCLUDED.content_status").
		Set("detected_kind = EXCLUDED.detected_kind").
		Set("shipment_draft_status = EXCLUDED.shipment_draft_status").
		Set("content_text = EXCLUDED.content_text").
		Set("created_at = EXCLUDED.created_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document search projection")
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	cols := buncolgen.ProjectionColumns
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*documentsearchprojection.Projection)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ProjectionScopeTenantDelete(dq, tenantInfo).
				Where(cols.ID.Eq(), documentID)
		}).
		Exec(ctx)
	if err != nil {
		return dberror.HandleNotFoundError(err, "Document search projection")
	}

	return nil
}
