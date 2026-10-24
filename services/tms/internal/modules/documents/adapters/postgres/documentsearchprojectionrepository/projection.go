package documentsearchprojectionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentsearchprojection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentsearchprojection/documentsearchprojectiondb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var projectionErrors = dberror.Mapping{
	Entity: "Document search projection",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_search_projections_document",
			Field:   "id",
			Message: "Document does not exist",
		},
	},
}

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
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentsearchprojection.Projection, error) {
			p := documentsearchprojectiondb.Projections
			_, err := p.Upsert(
				ctx,
				r.db.Keel(ctx),
				[]*documentsearchprojection.Projection{entity},
				keel.OnConflict(p.ID, p.OrganizationID, p.BusinessUnitID).Update(
					p.ResourceID,
					p.ResourceType,
					p.FileName,
					p.OriginalName,
					p.Description,
					p.Tags,
					p.Status,
					p.ContentStatus,
					p.DetectedKind,
					p.ShipmentDraftStatus,
					p.ContentText,
					p.CreatedAt,
					p.UpdatedAt,
				),
			)
			if err != nil {
				return nil, projectionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := documentsearchprojectiondb.Projections
		if _, err := p.Where(p.ID.Eq(documentID)).Delete(ctx, r.db.Keel(ctx)); err != nil {
			return projectionErrors.Map(err, documentID.String())
		}

		return nil
	})
}
