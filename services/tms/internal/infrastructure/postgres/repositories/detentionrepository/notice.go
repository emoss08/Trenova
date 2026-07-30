package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type NoticeParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type noticeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewNoticeRepository(p NoticeParams) repositories.DetentionNoticeRepository {
	return &noticeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-notice-repository"),
	}
}

func (r *noticeRepository) Create(
	ctx context.Context,
	entity *detention.DetentionNotice,
) (*detention.DetentionNotice, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create detention notice", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *noticeRepository) Update(
	ctx context.Context,
	entity *detention.DetentionNotice,
) (*detention.DetentionNotice, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update detention notice", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results, "DetentionNotice", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

// AttachPDFDocument records the stored notice PDF against the notice.
//
// The document is filed asynchronously, after the email has gone out, so this
// touches only pdf_document_id and updated_at: the delivery columns belong to
// whatever webhook lands next and must not be rolled back to the values this
// caller happened to read.
func (r *noticeRepository) AttachPDFDocument(
	ctx context.Context,
	req *repositories.AttachNoticePDFDocumentRequest,
) error {
	cols := buncolgen.DetentionNoticeColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*detention.DetentionNotice)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DetentionNoticeScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.NoticeID)
		}).
		Set(cols.PDFDocumentID.Set(), req.DocumentID).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to attach detention notice PDF document", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "DetentionNotice", req.NoticeID.String())
}

func (r *noticeRepository) List(
	ctx context.Context,
	req *repositories.ListDetentionNoticesRequest,
) ([]*detention.DetentionNotice, error) {
	cols := buncolgen.DetentionNoticeColumns
	entities := make([]*detention.DetentionNotice, 0, 8)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionNoticeScopeTenant(sq, req.TenantInfo).
				Where(cols.DetentionOccurrenceID.Eq(), req.OccurrenceID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list detention notices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// ListQueued drives the outbound sweep: notices whose scheduled send time has
// arrived but that have not left the building yet.
func (r *noticeRepository) ListQueued(
	ctx context.Context,
	req *repositories.ListNoticesDueRequest,
) ([]*detention.DetentionNotice, error) {
	cols := buncolgen.DetentionNoticeColumns
	entities := make([]*detention.DetentionNotice, 0, 32)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionNoticeScopeTenant(sq, req.TenantInfo).
				Where(cols.DeliveryStatus.Eq(), detention.NoticeDeliveryStatusQueued).
				Where(cols.ScheduledFor.Lte(), req.Before)
		}).
		Order(cols.ScheduledFor.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list queued detention notices", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
