package documentshipmentdraftrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.DocumentShipmentDraftRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-shipment-draft-repository"),
	}
}

func (r *repository) GetByDocumentID(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentshipmentdraft.Draft, error) {
	entity := new(documentshipmentdraft.Draft)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dsd.document_id = ?", documentID).
		Where("dsd.organization_id = ?", tenantInfo.OrgID).
		Where("dsd.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document shipment draft")
	}

	return entity, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *documentshipmentdraft.Draft,
) (*documentshipmentdraft.Draft, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(`CONFLICT ("document_id", "organization_id", "business_unit_id") DO UPDATE`).
		Set("status = EXCLUDED.status").
		Set("document_kind = EXCLUDED.document_kind").
		Set("confidence = EXCLUDED.confidence").
		Set("draft_data = EXCLUDED.draft_data").
		Set("failure_code = EXCLUDED.failure_code").
		Set("failure_message = EXCLUDED.failure_message").
		Set("attached_shipment_id = COALESCE(EXCLUDED.attached_shipment_id, dsd.attached_shipment_id)").
		Set("attached_at = COALESCE(EXCLUDED.attached_at, dsd.attached_at)").
		Set("attached_by_id = COALESCE(EXCLUDED.attached_by_id, dsd.attached_by_id)").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}
