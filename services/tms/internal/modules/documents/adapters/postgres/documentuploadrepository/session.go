package documentuploadrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload/documentuploaddb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultReconciliationLimit = 100

var sessionErrors = dberror.Mapping{Entity: "Document upload session"}

var (
	supersededStatuses = []documentupload.Status{
		documentupload.StatusInitiated,
		documentupload.StatusUploading,
		documentupload.StatusUploaded,
		documentupload.StatusVerifying,
		documentupload.StatusFinalizing,
		documentupload.StatusCompleting,
		documentupload.StatusPaused,
	}
	staleStatuses = []documentupload.Status{
		documentupload.StatusUploaded,
		documentupload.StatusVerifying,
		documentupload.StatusFinalizing,
		documentupload.StatusInitiated,
		documentupload.StatusUploading,
		documentupload.StatusPaused,
	}
	expirableStatuses = []documentupload.Status{
		documentupload.StatusInitiated,
		documentupload.StatusUploading,
		documentupload.StatusPaused,
	}
	terminalStatuses = []documentupload.Status{
		documentupload.StatusCompleted,
		documentupload.StatusAvailable,
		documentupload.StatusQuarantined,
		documentupload.StatusFailed,
		documentupload.StatusCanceled,
		documentupload.StatusExpired,
	}
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

func New(p Params) documentuploadports.DocumentUploadSessionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-upload-session-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *documentupload.DocumentUploadSession,
) (*documentupload.DocumentUploadSession, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentupload.DocumentUploadSession, error) {
			db := r.db.Keel(ctx)
			s := documentuploaddb.DocumentUploadSessions
			if err := s.Create(ctx, db, entity); err != nil {
				return nil, sessionErrors.Map(err, entity.ID.String())
			}

			if entity.LineageID == nil || entity.LineageID.IsNil() {
				return entity, nil
			}

			_, err := s.Where(
				s.LineageID.Eq(*entity.LineageID),
				s.ID.Ne(entity.ID),
				s.Status.In(supersededStatuses...),
			).Update(
				ctx,
				db,
				s.Status.To(documentupload.StatusCanceled),
				s.FailureCode.To(documentupload.FailureCodeSuspendedByNewerSession.String()),
				s.FailureMessage.To("Superseded by a newer upload session"),
				s.LastActivityAt.To(timeutils.NowUnix()),
			)
			if err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *documentupload.DocumentUploadSession,
) (*documentupload.DocumentUploadSession, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentupload.DocumentUploadSession, error) {
			err := documentuploaddb.DocumentUploadSessions.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, sessionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req documentuploadports.GetDocumentUploadSessionByIDRequest,
) (*documentupload.DocumentUploadSession, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentupload.DocumentUploadSession, error) {
			entity, err := documentuploaddb.DocumentUploadSessions.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, sessionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListForReconciliation(
	ctx context.Context,
	req *documentuploadports.ListDocumentUploadReconciliationRequest,
) ([]*documentupload.DocumentUploadSession, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documentupload.DocumentUploadSession, error) {
			s := documentuploaddb.DocumentUploadSessions
			return s.Where(needsReconciliation(s, req)).
				Order(s.LastActivityAt.Asc()).
				Limit(reconciliationLimit(req.Limit)).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *repository) ListReconciliationTenants(
	ctx context.Context,
	req *documentuploadports.ListDocumentUploadReconciliationRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with uploads to reconcile")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		type tenantRow struct {
			OrganizationID pulid.ID
			BusinessUnitID pulid.ID
		}

		s := documentuploaddb.DocumentUploadSessions
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), s.AllTenants().
			Where(needsReconciliation(s, req)).
			Group(s.OrganizationID, s.BusinessUnitID).
			Pick(s.OrganizationID, s.BusinessUnitID).
			Order(s.OrganizationID.Asc(), s.BusinessUnitID.Asc()).
			Limit(reconciliationLimit(req.Limit)))
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
	})
}

func (r *repository) ClearDocumentReference(
	ctx context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := documentuploaddb.DocumentUploadSessions
		_, err := s.Where(s.DocumentID.Eq(documentID)).
			Update(ctx, r.db.Keel(ctx), detachDocument()...)

		return err
	})
}

func (r *repository) ClearDocumentReferences(
	ctx context.Context,
	documentIDs []pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(documentIDs) == 0 {
			return nil
		}

		s := documentuploaddb.DocumentUploadSessions
		_, err := s.Where(s.DocumentID.In(documentIDs...)).
			Update(ctx, r.db.Keel(ctx), detachDocument()...)

		return err
	})
}

func detachDocument() []keel.Assignment {
	s := documentuploaddb.DocumentUploadSessions
	return []keel.Assignment{
		s.DocumentID.ToNull(),
		s.Version.ToExpr(s.Version),
		s.UpdatedAt.ToExpr(s.UpdatedAt),
	}
}

func (r *repository) ListActive(
	ctx context.Context,
	req *documentuploadports.ListActiveDocumentUploadSessionsRequest,
) ([]*documentupload.DocumentUploadSession, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documentupload.DocumentUploadSession, error) {
			s := documentuploaddb.DocumentUploadSessions
			sessions, err := s.Where(
				s.Status.NotIn(terminalStatuses...),
				keel.When(req.ResourceID != "", s.ResourceID.Eq(req.ResourceID)),
				keel.When(req.ResourceType != "", s.ResourceType.Eq(req.ResourceType)),
			).
				Order(s.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			return orEmpty(sessions), nil
		},
	)
}

func needsReconciliation(
	s *documentuploaddb.DocumentUploadSessionsTable,
	req *documentuploadports.ListDocumentUploadReconciliationRequest,
) keel.Pred {
	return keel.Or(
		keel.And(s.Status.In(staleStatuses...), s.LastActivityAt.Lte(req.StaleBefore)),
		keel.And(s.Status.In(expirableStatuses...), s.ExpiresAt.Lte(req.ExpiresBefore)),
	)
}

func reconciliationLimit(limit int) int {
	if limit <= 0 {
		return defaultReconciliationLimit
	}

	return limit
}

func orEmpty(
	items []*documentupload.DocumentUploadSession,
) []*documentupload.DocumentUploadSession {
	if items == nil {
		return []*documentupload.DocumentUploadSession{}
	}

	return items
}
