package documentuploadrepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.DocumentUploadSessionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-upload-session-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentupload.Session,
) (*documentupload.Session, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{Isolation: 0}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			return err
		}

		if entity.LineageID == nil || entity.LineageID.IsNil() {
			return nil
		}

		_, err := tx.NewUpdate().
			Table("document_upload_sessions").
			Set("status = ?", documentupload.StatusCanceled).
			Set("failure_code = ?", "SUPERSEDED_BY_NEWER_SESSION").
			Set("failure_message = ?", "Superseded by a newer upload session").
			Set("last_activity_at = ?", time.Now().Unix()).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Where("lineage_id = ?", *entity.LineageID).
			Where("id <> ?", entity.ID).
			Where("status IN (?)", bun.In([]documentupload.Status{
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
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentupload.Session,
) (*documentupload.Session, error) {
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

	if err = dberror.CheckRowsAffected(result, "Document upload session", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDocumentUploadSessionByIDRequest,
) (*documentupload.Session, error) {
	entity := new(documentupload.Session)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dus.id = ?", req.ID).
				Where("dus.organization_id = ?", req.TenantInfo.OrgID).
				Where("dus.business_unit_id = ?", req.TenantInfo.BuID)
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
) ([]*documentupload.Session, error) {
	sessions := make([]*documentupload.Session, 0, limit)
	if limit <= 0 {
		limit = 100
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&sessions).
		WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereGroup(" AND ", func(active *bun.SelectQuery) *bun.SelectQuery {
				return active.
					Where("dus.status IN (?)", bun.In([]documentupload.Status{
						documentupload.StatusUploaded,
						documentupload.StatusVerifying,
						documentupload.StatusFinalizing,
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where("dus.last_activity_at <= ?", staleBefore)
			}).WhereGroup(" AND ", func(expired *bun.SelectQuery) *bun.SelectQuery {
				return expired.
					Where("dus.status IN (?)", bun.In([]documentupload.Status{
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where("dus.expires_at <= ?", expiresBefore)
			})
		}).
		Order("dus.last_activity_at ASC").
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
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Table("document_upload_sessions").
		Set("document_id = NULL").
		Where("document_id = ?", documentID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
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

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Table("document_upload_sessions").
		Set("document_id = NULL").
		Where("document_id IN (?)", bun.In(documentIDs)).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	return err
}

func (r *repository) ListActive(
	ctx context.Context,
	req *repositories.ListActiveDocumentUploadSessionsRequest,
) ([]*documentupload.Session, error) {
	sessions := make([]*documentupload.Session, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&sessions).
		Where("dus.organization_id = ?", req.TenantInfo.OrgID).
		Where("dus.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dus.status NOT IN (?)", bun.In([]documentupload.Status{
			documentupload.StatusCompleted,
			documentupload.StatusAvailable,
			documentupload.StatusQuarantined,
			documentupload.StatusFailed,
			documentupload.StatusCanceled,
			documentupload.StatusExpired,
		}))

	if req.ResourceID != "" {
		q = q.Where("dus.resource_id = ?", req.ResourceID)
	}

	if req.ResourceType != "" {
		q = q.Where("dus.resource_type = ?", req.ResourceType)
	}

	if err := q.Order("dus.created_at DESC").Scan(ctx); err != nil {
		return nil, err
	}

	return sessions, nil
}
