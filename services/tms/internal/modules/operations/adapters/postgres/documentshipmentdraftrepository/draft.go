package documentshipmentdraftrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft/documentshipmentdraftdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/documentshipmentdraftports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var draftErrors = dberror.Mapping{Entity: "Document shipment draft"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) documentshipmentdraftports.DocumentShipmentDraftRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-shipment-draft-repository"),
	}
}

func (r *repository) GetByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			d := documentshipmentdraftdb.DocumentShipmentDrafts
			entity, err := d.Where(d.DocumentID.Eq(documentID)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, draftErrors.Map(err, documentID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentshipmentdraft.DocumentShipmentDraft,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentshipmentdraft.DocumentShipmentDraft, error) {
			d := documentshipmentdraftdb.DocumentShipmentDrafts
			db := r.db.Keel(ctx)
			inserted, err := d.Upsert(
				ctx,
				db,
				[]*documentshipmentdraft.DocumentShipmentDraft{entity},
				keel.OnConflict(d.DocumentID, d.OrganizationID, d.BusinessUnitID).
					Update(
						d.Status,
						d.DocumentKind,
						d.Confidence,
						d.DraftData,
						d.FailureCode,
						d.FailureMessage,
					).
					Set(
						d.AttachedShipmentID.ToExpr(
							keel.DynOf(d.AttachedShipmentID.Excluded()).Coalesce(d.AttachedShipmentID),
						),
						d.AttachedAt.ToExpr(
							keel.DynOf(d.AttachedAt.Excluded()).Coalesce(d.AttachedAt),
						),
						d.AttachedByID.ToExpr(
							keel.DynOf(d.AttachedByID.Excluded()).Coalesce(d.AttachedByID),
						),
					),
			)
			if err != nil {
				r.l.Error("failed to upsert document shipment draft", zap.Error(err))
				return nil, draftErrors.Map(err, entity.ID.String())
			}
			if len(inserted) == 1 && inserted[0] {
				return entity, nil
			}

			stored, err := d.Where(d.DocumentID.Eq(entity.DocumentID)).One(ctx, db)
			if err != nil {
				return nil, draftErrors.Map(err, entity.DocumentID.String())
			}

			return stored, nil
		},
	)
}
