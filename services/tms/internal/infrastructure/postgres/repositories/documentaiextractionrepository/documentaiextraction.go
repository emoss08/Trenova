package documentaiextractionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.DocumentAIExtractionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-ai-extraction-repository"),
	}
}

func (r *repository) GetByDocumentExtractedAt(
	ctx context.Context,
	req repositories.GetDocumentAIExtractionRequest,
) (*documentaiextraction.Extraction, error) {
	entity := new(documentaiextraction.Extraction)
	cols := buncolgen.ExtractionColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionScopeTenant(sq, req.TenantInfo).
				Where(cols.DocumentID.Eq(), req.DocumentID).
				Where(cols.ExtractedAt.Eq(), req.ExtractedAt)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DocumentAIExtraction")
	}

	return entity, nil
}

func (r *repository) SavePending(
	ctx context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	cols := buncolgen.ExtractionColumns
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("document_id", "organization_id", "business_unit_id", "extracted_at") DO UPDATE`).
		Set(cols.UserID.SetExcluded()).
		Set(cols.RequestHash.SetExcluded()).
		Set(cols.WorkflowID.SetExcluded()).
		Set(cols.WorkflowRunID.SetExcluded()).
		Set(cols.ActivityID.SetExcluded()).
		Set(cols.TaskToken.SetExcluded()).
		Set(cols.ResponseID.SetExpr("COALESCE(NULLIF(EXCLUDED.response_id, ''), dae.response_id)")).
		Set(cols.Model.SetExpr("COALESCE(NULLIF(EXCLUDED.model, ''), dae.model)")).
		Set(cols.Status.SetExpr("CASE WHEN dae.status IN (?, ?) THEN dae.status ELSE EXCLUDED.status END"), documentaiextraction.StatusApplied, documentaiextraction.StatusSkipped).
		Set(cols.SubmittedAt.SetExpr("COALESCE(EXCLUDED.submitted_at, dae.submitted_at)")).
		Set(cols.FailureCode.SetExcluded()).
		Set(cols.FailureMessage.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentaiextraction.Extraction,
) (*documentaiextraction.Extraction, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.ExtractionColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		result,
		"DocumentAIExtraction",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListPollable(
	ctx context.Context,
	olderThan int64,
	limit int,
) ([]*documentaiextraction.Extraction, error) {
	if limit <= 0 {
		limit = 50
	}

	items := make([]*documentaiextraction.Extraction, 0, limit)
	cols := buncolgen.ExtractionColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where(cols.Status.Eq(), documentaiextraction.StatusPending).
		Where(cols.ResponseID.NotEq(), "").
		WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.LastPolledAt.IsNull()).
				WhereOr(cols.LastPolledAt.Lte(), olderThan)
		}).
		OrderExpr("COALESCE(dae.last_polled_at, 0) ASC").
		Order(cols.CreatedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	return items, err
}
