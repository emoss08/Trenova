package documentaiextractionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction/documentaiextractiondb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var extractionErrors = dberror.Mapping{
	Entity: "DocumentAIExtraction",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_ai_extractions_document",
			Field:   "documentId",
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

func New(p Params) documentaiextractionports.DocumentAIExtractionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-ai-extraction-repository"),
	}
}

func (r *repository) GetByDocumentExtractedAt(
	ctx context.Context,
	req documentaiextractionports.GetDocumentAIExtractionRequest,
) (*documentaiextraction.Extraction, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentaiextraction.Extraction, error) {
			entity, err := byDocumentExtractedAt(req.DocumentID, req.ExtractedAt).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, extractionErrors.Map(err, req.DocumentID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) SavePending(
	ctx context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentaiextraction.Extraction, error) {
			ex := documentaiextractiondb.Extractions
			db := r.db.Keel(ctx)
			inserted, err := ex.Upsert(
				ctx,
				db,
				[]*documentaiextraction.Extraction{entity},
				keel.OnConflict(ex.DocumentID, ex.OrganizationID, ex.BusinessUnitID, ex.ExtractedAt).
					Update(
						ex.UserID,
						ex.RequestHash,
						ex.WorkflowID,
						ex.WorkflowRunID,
						ex.ActivityID,
						ex.FailureCode,
						ex.FailureMessage,
					).
					Set(
						ex.ResponseID.ToExpr(keel.DynOf(ex.ResponseID.Excluded()).NullIf("").Coalesce(ex.ResponseID)),
						ex.Model.ToExpr(keel.DynOf(ex.Model.Excluded()).NullIf("").Coalesce(ex.Model)),
						ex.Status.ToExpr(keel.Raw(
							"CASE WHEN ? IN (?, ?) THEN ? ELSE ? END",
							ex.Status,
							string(documentaiextraction.StatusApplied),
							string(documentaiextraction.StatusSkipped),
							ex.Status,
							ex.Status.Excluded(),
						)),
						ex.SubmittedAt.ToExpr(keel.DynOf(ex.SubmittedAt.Excluded()).Coalesce(ex.SubmittedAt)),
					),
			)
			if err != nil {
				return nil, extractionErrors.Map(err, entity.ID.String())
			}
			if len(inserted) == 1 && inserted[0] {
				return entity, nil
			}

			stored, err := byDocumentExtractedAt(entity.DocumentID, entity.ExtractedAt).One(ctx, db)
			if err != nil {
				return nil, extractionErrors.Map(err, entity.DocumentID.String())
			}
			*entity = *stored

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentaiextraction.Extraction, error) {
			err := documentaiextractiondb.Extractions.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, extractionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func byDocumentExtractedAt(
	documentID pulid.ID,
	extractedAt int64,
) *keel.Query[documentaiextraction.Extraction] {
	ex := documentaiextractiondb.Extractions
	return ex.Where(ex.DocumentID.Eq(documentID), ex.ExtractedAt.Eq(extractedAt))
}
