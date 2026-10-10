package documentsearchprojectionrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/documentsearchprojectioncol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
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

func New(p Params) documentsearchprojectionports.DocumentSearchProjectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-search-projection-repository"),
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentsearchprojection.Projection,
) (*documentsearchprojection.Projection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentsearchprojection.Projection, error) {
		cols := documentsearchprojectioncol.ProjectionColumns
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On(`CONFLICT ("id", "organization_id", "business_unit_id") DO UPDATE`).
			Set(cols.ResourceID.SetExcluded()).
			Set(cols.ResourceType.SetExcluded()).
			Set(cols.FileName.SetExcluded()).
			Set(cols.OriginalName.SetExcluded()).
			Set(cols.Description.SetExcluded()).
			Set(cols.Tags.SetExcluded()).
			Set(cols.Status.SetExcluded()).
			Set(cols.ContentStatus.SetExcluded()).
			Set(cols.DetectedKind.SetExcluded()).
			Set(cols.ShipmentDraftStatus.SetExcluded()).
			Set(cols.ContentText.SetExcluded()).
			Set(cols.CreatedAt.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Document search projection")
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := documentsearchprojectioncol.ProjectionColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*documentsearchprojection.Projection)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return documentsearchprojectioncol.ProjectionScopeTenantDelete(dq, tenantInfo).
					Where(cols.ID.Eq(), documentID)
			}).
			Exec(ctx)
		if err != nil {
			return dberror.HandleNotFoundError(err, "Document search projection")
		}

		return nil
	})
}
