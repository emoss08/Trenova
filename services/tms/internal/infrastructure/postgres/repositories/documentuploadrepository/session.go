package documentuploadrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.DocumentUploadSessionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-upload-session-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentupload.DocumentUploadSession,
) (*documentupload.DocumentUploadSession, error) {
	cols := buncolgen.DocumentUploadSessionColumns
	err := r.db.WithTx(
		ctx,
		ports.TxOptions{Isolation: 0},
		func(c context.Context, tx bun.Tx) error {
			if _, err := tx.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
				return err
			}

			if entity.LineageID == nil || entity.LineageID.IsNil() {
				return nil
			}

			_, err := tx.NewUpdate().
				Table(buncolgen.DocumentUploadSessionTable.Name).
				Set(cols.Status.Set(), documentupload.StatusCanceled).
				Set(cols.FailureCode.Set(), documentupload.FailureCodeSuspendedByNewerSession).
				Set(cols.FailureMessage.Set(), "Superseded by a newer upload session").
				Set(cols.LastActivityAt.Set(), timeutils.NowUnix()).
				Where(cols.OrganizationID.Eq(), entity.OrganizationID).
				Where(cols.BusinessUnitID.Eq(), entity.BusinessUnitID).
				Where(cols.LineageID.Eq(), *entity.LineageID).
				Where(cols.ID.NotEq(), entity.ID).
				Where(cols.Status.In(), bun.List([]documentupload.Status{
					documentupload.StatusInitiated,
					documentupload.StatusUploading,
					documentupload.StatusUploaded,
					documentupload.StatusVerifying,
					documentupload.StatusFinalizing,
					documentupload.StatusCompleting,
					documentupload.StatusPaused,
				})).
				Exec(ctx)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentupload.DocumentUploadSession,
) (*documentupload.DocumentUploadSession, error) {
	ov := entity.Version
	entity.Version++

	cols := buncolgen.DocumentUploadSessionColumns
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
		"Document upload session",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentUploadSessionByIDRequest,
) (*documentupload.DocumentUploadSession, error) {
	entity := new(documentupload.DocumentUploadSession)
	cols := buncolgen.DocumentUploadSessionColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentUploadSessionScopeTenant(
				sq,
				req.TenantInfo,
			).Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Document upload session")
	}

	return entity, nil
}

func (r *repository) ListForReconciliation(
	ctx context.Context,
	staleBefore int64,
	expiresBefore int64,
	limit int,
) ([]*documentupload.DocumentUploadSession, error) {
	sessions := make([]*documentupload.DocumentUploadSession, 0, limit)
	if limit <= 0 {
		limit = 100
	}

	cols := buncolgen.DocumentUploadSessionColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&sessions).
		WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereGroup(" AND ", func(active *bun.SelectQuery) *bun.SelectQuery {
				return active.
					Where(cols.Status.In(), bun.List([]documentupload.Status{
						documentupload.StatusUploaded,
						documentupload.StatusVerifying,
						documentupload.StatusFinalizing,
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where(cols.LastActivityAt.Lte(), staleBefore)
			}).WhereGroup(" AND ", func(expired *bun.SelectQuery) *bun.SelectQuery {
				return expired.
					Where(cols.Status.In(), bun.List([]documentupload.Status{
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where(cols.ExpiresAt.Lte(), expiresBefore)
			})
		}).
		Order(cols.LastActivityAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *repository) ClearDocumentReference(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	cols := buncolgen.DocumentUploadSessionColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Table(buncolgen.DocumentUploadSessionTable.Name).
		Set(cols.DocumentID.Set(), nil).
		Where(cols.DocumentID.Eq(), documentID).
		Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Exec(ctx)
	return err
}

func (r *repository) ClearDocumentReferences(
	ctx context.Context,
	documentIDs []pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if len(documentIDs) == 0 {
		return nil
	}

	cols := buncolgen.DocumentUploadSessionColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Table(buncolgen.DocumentUploadSessionTable.Name).
		Set(cols.DocumentID.Set(), nil).
		Where(cols.DocumentID.In(), bun.List(documentIDs)).
		Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Exec(ctx)
	return err
}

func (r *repository) ListActive(
	ctx context.Context,
	req *repositories.ListActiveDocumentUploadSessionsRequest,
) ([]*documentupload.DocumentUploadSession, error) {
	sessions := make([]*documentupload.DocumentUploadSession, 0)
	cols := buncolgen.DocumentUploadSessionColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&sessions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentUploadSessionScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.NotIn(), bun.List([]documentupload.Status{
					documentupload.StatusCompleted,
					documentupload.StatusAvailable,
					documentupload.StatusQuarantined,
					documentupload.StatusFailed,
					documentupload.StatusCanceled,
					documentupload.StatusExpired,
				}))
		})

	if req.ResourceID != "" {
		q = q.Where(cols.ResourceID.Eq(), req.ResourceID)
	}

	if req.ResourceType != "" {
		q = q.Where(cols.ResourceType.Eq(), req.ResourceType)
	}

	if err := q.Order(cols.CreatedAt.OrderDesc()).Scan(ctx); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *repository) ListRelated(
	ctx context.Context,
	req *repositories.ListRelatedDocumentUploadSessionsRequest,
) ([]*documentupload.DocumentUploadSession, error) {
	sessions := make([]*documentupload.DocumentUploadSession, 0)
	cols := buncolgen.DocumentUploadSessionColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&sessions).Apply(buncolgen.DocumentUploadSessionApplyTenant(req.TenantInfo))

	switch {
	case !req.DocumentID.IsNil() && !req.LineageID.IsNil():
		q = q.WhereOr(cols.DocumentID.Eq(), req.DocumentID).
			WhereOr(cols.LineageID.Eq(), req.LineageID)
	case !req.DocumentID.IsNil():
		q = q.Where(cols.DocumentID.Eq(), req.DocumentID)
	case !req.LineageID.IsNil():
		q = q.Where(cols.LineageID.Eq(), req.LineageID)
	default:
		return sessions, nil
	}

	if err := q.Order(cols.CreatedAt.OrderDesc()).Scan(ctx); err != nil {
		return nil, err
	}

	return sessions, nil
}
