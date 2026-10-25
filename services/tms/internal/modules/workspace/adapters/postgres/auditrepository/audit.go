package auditrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var entryErrors = dberror.Mapping{Entity: "Audit Entry"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.AuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-repository"),
	}
}

func withPrincipals(q *keel.Query[*audit.Entry]) *keel.Query[*audit.Entry] {
	e := auditdb.Entries
	return q.Load(e.User(), e.APIKey())
}

func (r *repository) List(
	ctx context.Context,
	req *auditports.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*audit.Entry], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				auditdb.Entries.Query(),
				&listquery.Request[audit.Entry]{Filter: req.Filter, Shape: withPrincipals},
			)
			if err != nil {
				r.l.Error("failed to list audit entries", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *auditports.ListAuditEntriesConnectionRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*audit.Entry], error) {
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: "timestamp", Direction: dbtype.SortDirectionDesc},
				}
			}

			e := auditdb.Entries
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				e.Where(
					keel.When(!req.ResourceID.IsNil(), e.ResourceID.Eq(req.ResourceID.String())),
				),
				&listquery.Request[audit.Entry]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPrincipals,
				},
			)
			if err != nil {
				r.l.Error("failed to list audit entry connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.Entry, error) {
		e := auditdb.Entries
		entity, err := withPrincipals(e.Where(e.ID.Eq(req.EntryID))).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get audit entry",
				zap.String("entryID", req.EntryID.String()),
				zap.Error(err),
			)
			return nil, entryErrors.Map(err, req.EntryID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByResourceID(
	ctx context.Context,
	req *auditports.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*audit.Entry], error) {
			e := auditdb.Entries
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				e.Where(e.ResourceID.Eq(req.ResourceID.String())),
				&listquery.Request[audit.Entry]{Filter: req.Filter, Shape: withPrincipals},
			)
			if err != nil {
				r.l.Error(
					"failed to list audit entries by resource",
					zap.String("resourceID", req.ResourceID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error {
	ctx = dbscope.WithSystem(
		ctx,
		"flush buffered audit entries, which mix organizations, in one batch",
	)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := auditdb.Entries.CreateMany(ctx, r.db.Keel(ctx), entries); err != nil {
			r.l.Error("failed to insert audit entries", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) DeleteAuditEntries(
	ctx context.Context,
	req auditports.DeleteAuditEntriesRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete audit entries past an organization's retention during the retention sweep",
	)
	return postgres.DeleteUnderAuditRetention(
		ctx,
		r.db,
		func(ctx context.Context, db *keel.DB) (int64, error) {
			e := auditdb.Entries
			return e.AllTenants().Where(
				e.OrganizationID.Eq(req.OrgID),
				e.BusinessUnitID.Eq(req.BuID),
				e.Timestamp.Lt(req.Before),
				keel.Or(e.Critical.IsFalse(), e.Timestamp.Lt(req.CriticalBefore)),
			).Delete(ctx, db)
		},
	)
}
