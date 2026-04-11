package journalsourcerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.JournalSourceRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-source-repository")}
}

type sourceRecord struct {
	bun.BaseModel `bun:"table:journal_sources,alias:js"`

	ID                   string `bun:"id,pk"`
	OrganizationID       string `bun:"organization_id,pk"`
	BusinessUnitID       string `bun:"business_unit_id,pk"`
	SourceObjectType     string `bun:"source_object_type"`
	SourceObjectID       string `bun:"source_object_id"`
	SourceEventType      string `bun:"source_event_type"`
	SourceDocumentNumber string `bun:"source_document_number"`
	Status               string `bun:"status"`
	IdempotencyKey       string `bun:"idempotency_key"`
	JournalBatchID       string `bun:"journal_batch_id"`
	JournalEntryID       string `bun:"journal_entry_id"`
}

func (r *repository) GetByObject(ctx context.Context, req repositories.GetJournalSourceByObjectRequest) (*journalsource.Source, error) {
	return r.getByObjectQuery(ctx, req, "")
}

func (r *repository) GetByObjectAndEvent(ctx context.Context, req repositories.GetJournalSourceByObjectRequest, sourceEventType string) (*journalsource.Source, error) {
	return r.getByObjectQuery(ctx, req, sourceEventType)
}

func (r *repository) getByObjectQuery(ctx context.Context, req repositories.GetJournalSourceByObjectRequest, sourceEventType string) (*journalsource.Source, error) {
	rec := new(sourceRecord)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(rec).
		Where("js.organization_id = ?", req.TenantInfo.OrgID).
		Where("js.business_unit_id = ?", req.TenantInfo.BuID).
		Where("js.source_object_type = ?", req.SourceObjectType).
		Where("js.source_object_id = ?", req.SourceObjectID).
		Order("js.created_at DESC").
		Limit(1)
	if sourceEventType != "" {
		query = query.Where("js.source_event_type = ?", sourceEventType)
	}
	err := query.Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "JournalSource")
	}
	return &journalsource.Source{ID: pulid.ID(rec.ID), OrganizationID: pulid.ID(rec.OrganizationID), BusinessUnitID: pulid.ID(rec.BusinessUnitID), SourceObjectType: rec.SourceObjectType, SourceObjectID: rec.SourceObjectID, SourceEventType: rec.SourceEventType, SourceDocumentNumber: rec.SourceDocumentNumber, Status: rec.Status, IdempotencyKey: rec.IdempotencyKey, JournalBatchID: pulid.ID(rec.JournalBatchID), JournalEntryID: pulid.ID(rec.JournalEntryID)}, nil
}
