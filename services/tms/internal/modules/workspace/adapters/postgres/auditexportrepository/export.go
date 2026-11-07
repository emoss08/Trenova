package auditexportrepository

import (
	"context"

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

var exportErrors = dberror.Mapping{Entity: "Audit export"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.ExportRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-export-repository")}
}

func (r *repository) Create(ctx context.Context, export *audit.Export) (*audit.Export, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.Export, error) {
		if err := auditdb.Exports.Create(ctx, r.db.Keel(ctx), export); err != nil {
			return nil, exportErrors.Map(err, export.ID.String())
		}
		return export, nil
	})
}

func (r *repository) Update(ctx context.Context, export *audit.Export) (*audit.Export, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.Export, error) {
		if err := auditdb.Exports.Save(ctx, r.db.Keel(ctx), export); err != nil {
			return nil, exportErrors.Map(err, export.ID.String())
		}
		return export, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req auditports.GetExportRequest,
) (*audit.Export, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.Export, error) {
		export, err := auditdb.Exports.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, exportErrors.Map(err, req.ID.String())
		}
		return export, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *auditports.ListExportsRequest,
) (*pagination.CursorListResult[*audit.Export], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*audit.Export], error) {
			x := auditdb.Exports
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: "createdAt", Direction: dbtype.SortDirectionDesc},
				}
			}
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				x.Where(x.RequestedByID.Eq(req.RequestedByID)),
				&listquery.Request[audit.Export]{Filter: req.Filter, Cursor: req.Cursor},
			)
		},
	)
}

func (r *repository) ListExpired(
	ctx context.Context,
	now int64,
	limit int,
) ([]*audit.Export, error) {
	ctx = dbscope.WithSystem(ctx, "expire every organization's audit exports past their download window")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.Export, error) {
		x := auditdb.Exports
		return x.AllTenants().
			Where(x.ArtifactKey.IsNotNull(), x.ArtifactExpiresAt.Lte(now)).
			Order(x.ArtifactExpiresAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}
