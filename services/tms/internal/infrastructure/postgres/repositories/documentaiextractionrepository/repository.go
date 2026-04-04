package documentaiextractionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dae.document_id = ?", req.DocumentID).
		Where("dae.organization_id = ?", req.TenantInfo.OrgID).
		Where("dae.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dae.extracted_at = ?", req.ExtractedAt).
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
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("document_id", "organization_id", "business_unit_id", "extracted_at") DO UPDATE`).
		Set("user_id = EXCLUDED.user_id").
		Set("request_hash = EXCLUDED.request_hash").
		Set("workflow_id = EXCLUDED.workflow_id").
		Set("workflow_run_id = EXCLUDED.workflow_run_id").
		Set("activity_id = EXCLUDED.activity_id").
		Set("task_token = EXCLUDED.task_token").
		Set("response_id = COALESCE(NULLIF(EXCLUDED.response_id, ''), dae.response_id)").
		Set("model = COALESCE(NULLIF(EXCLUDED.model, ''), dae.model)").
		Set("status = CASE WHEN dae.status IN (?, ?) THEN dae.status ELSE EXCLUDED.status END", documentaiextraction.StatusApplied, documentaiextraction.StatusSkipped).
		Set("submitted_at = COALESCE(EXCLUDED.submitted_at, dae.submitted_at)").
		Set("failure_code = EXCLUDED.failure_code").
		Set("failure_message = EXCLUDED.failure_message").
		Set("updated_at = EXCLUDED.updated_at").
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
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "DocumentAIExtraction", entity.ID.String()); err != nil {
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dae.status = ?", documentaiextraction.StatusPending).
		Where("dae.response_id <> ''").
		Where("(dae.last_polled_at IS NULL OR dae.last_polled_at <= ?)", olderThan).
		OrderExpr("COALESCE(dae.last_polled_at, 0) ASC, dae.created_at ASC").
		Limit(limit).
		Scan(ctx)
	return items, err
}
