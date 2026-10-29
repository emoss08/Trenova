package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice/detentionnoticedb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
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

func NewNoticeRepository(p NoticeParams) detentionports.DetentionNoticeRepository {
	return &noticeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-notice-repository"),
	}
}

func (r *noticeRepository) Create(
	ctx context.Context,
	entity *detentionnotice.DetentionNotice,
) (*detentionnotice.DetentionNotice, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionnotice.DetentionNotice, error) {
			if err := detentionnoticedb.DetentionNotices.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to create detention notice", zap.Error(err))
				return nil, noticeErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *noticeRepository) Update(
	ctx context.Context,
	entity *detentionnotice.DetentionNotice,
) (*detentionnotice.DetentionNotice, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionnotice.DetentionNotice, error) {
			if err := detentionnoticedb.DetentionNotices.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error("failed to update detention notice", zap.Error(err))
				return nil, noticeErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// AttachPDFDocument records the stored notice PDF against the notice.
//
// The document is filed asynchronously, after the email has gone out, so this
// touches only pdf_document_id and updated_at: the delivery columns belong to
// whatever webhook lands next and must not be rolled back to the values this
// caller happened to read.
func (r *noticeRepository) AttachPDFDocument(
	ctx context.Context,
	req *detentionports.AttachNoticePDFDocumentRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := detentionnoticedb.DetentionNotices
		rows, err := n.Where(n.ID.Eq(req.NoticeID)).Update(
			ctx,
			r.db.Keel(ctx),
			n.PDFDocumentID.To(req.DocumentID),
			n.UpdatedAt.To(timeutils.NowUnix()),
			n.Version.ToExpr(n.Version),
		)
		if err != nil {
			r.l.Error("failed to attach detention notice PDF document", zap.Error(err))
			return noticeErrors.Map(err, req.NoticeID.String())
		}

		return dberror.CheckAffected(rows, "DetentionNotice", req.NoticeID.String())
	})
}

func (r *noticeRepository) List(
	ctx context.Context,
	req *detentionports.ListDetentionNoticesRequest,
) ([]*detentionnotice.DetentionNotice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionnotice.DetentionNotice, error) {
			n := detentionnoticedb.DetentionNotices
			entities, err := n.Where(n.DetentionOccurrenceID.Eq(req.OccurrenceID)).
				Order(n.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention notices", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

// ListQueued drives the outbound sweep: notices whose scheduled send time has
// arrived but that have not left the building yet.
func (r *noticeRepository) ListQueued(
	ctx context.Context,
	req *detentionports.ListNoticesDueRequest,
) ([]*detentionnotice.DetentionNotice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionnotice.DetentionNotice, error) {
			n := detentionnoticedb.DetentionNotices
			entities, err := n.Where(
				n.DeliveryStatus.Eq(detentionnotice.NoticeDeliveryStatusQueued),
				n.ScheduledFor.Lte(req.Before),
			).
				Order(n.ScheduledFor.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list queued detention notices", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}
