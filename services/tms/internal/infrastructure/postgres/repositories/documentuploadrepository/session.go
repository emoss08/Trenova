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
				Where("? = ?", bun.Ident(cols.OrganizationID.Name), entity.OrganizationID).
				Where("? = ?", bun.Ident(cols.BusinessUnitID.Name), entity.BusinessUnitID).
				Where("? = ?", bun.Ident(cols.LineageID.Name), *entity.LineageID).
				Where("? <> ?", bun.Ident(cols.ID.Name), entity.ID).
				Where("? IN (?)", bun.Ident(cols.Status.Name), bun.List([]documentupload.Status{
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
	req *repositories.ListDocumentUploadReconciliationRequest,
) ([]*documentupload.DocumentUploadSession, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	sessions := make([]*documentupload.DocumentUploadSession, 0, limit)

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
					Where(cols.LastActivityAt.Lte(), req.StaleBefore)
			}).WhereGroup(" AND ", func(expired *bun.SelectQuery) *bun.SelectQuery {
				return expired.
					Where(cols.Status.In(), bun.List([]documentupload.Status{
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where(cols.ExpiresAt.Lte(), req.ExpiresBefore)
			})
		}).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Order(cols.LastActivityAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *repository) ListReconciliationTenants(
	ctx context.Context,
	req *repositories.ListDocumentUploadReconciliationRequest,
) ([]pagination.TenantInfo, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	type tenantRow struct {
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}

	rows := make([]tenantRow, 0, limit)
	cols := buncolgen.DocumentUploadSessionColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*documentupload.DocumentUploadSession)(nil)).
		Column(cols.OrganizationID.Name, cols.BusinessUnitID.Name).
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
					Where(cols.LastActivityAt.Lte(), req.StaleBefore)
			}).WhereGroup(" AND ", func(expired *bun.SelectQuery) *bun.SelectQuery {
				return expired.
					Where(cols.Status.In(), bun.List([]documentupload.Status{
						documentupload.StatusInitiated,
						documentupload.StatusUploading,
						documentupload.StatusPaused,
					})).
					Where(cols.ExpiresAt.Lte(), req.ExpiresBefore)
			})
		}).
		Group(cols.OrganizationID.Name, cols.BusinessUnitID.Name).
		Order(cols.OrganizationID.OrderAsc()).
		Order(cols.BusinessUnitID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, len(rows))
	for _, row := range rows {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: row.OrganizationID,
			BuID:  row.BusinessUnitID,
		})
	}

	return tenants, nil
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
		Where("? = ?", bun.Ident(cols.DocumentID.Name), documentID).
		Where("? = ?", bun.Ident(cols.OrganizationID.Name), tenantInfo.OrgID).
		Where("? = ?", bun.Ident(cols.BusinessUnitID.Name), tenantInfo.BuID).
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
		Where("? IN (?)", bun.Ident(cols.DocumentID.Name), bun.List(documentIDs)).
		Where("? = ?", bun.Ident(cols.OrganizationID.Name), tenantInfo.OrgID).
		Where("? = ?", bun.Ident(cols.BusinessUnitID.Name), tenantInfo.BuID).
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
