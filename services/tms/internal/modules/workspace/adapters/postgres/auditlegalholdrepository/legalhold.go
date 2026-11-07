package auditlegalholdrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var holdErrors = dberror.Mapping{Entity: "Legal hold"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.LegalHoldRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-legal-hold-repository")}
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *auditports.ListLegalHoldsRequest,
) (*pagination.CursorListResult[*audit.LegalHold], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*audit.LegalHold], error) {
			h := auditdb.LegalHolds
			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: "placedAt", Direction: dbtype.SortDirectionDesc},
				}
			}
			q := h.Query()
			if req.ActiveOnly {
				q = h.Where(h.ReleasedAt.IsNull())
			}
			return listquery.Connection(ctx, r.db.Keel(ctx), q, &listquery.Request[audit.LegalHold]{
				Filter: req.Filter,
				Cursor: req.Cursor,
			})
		},
	)
}

func (r *repository) ListActive(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*audit.LegalHold, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.LegalHold, error) {
		h := auditdb.LegalHolds
		return h.Where(h.ReleasedAt.IsNull()).
			Order(h.PlacedAt.Desc()).
			Limit(1000).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req auditports.GetLegalHoldRequest,
) (*audit.LegalHold, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.LegalHold, error) {
		hold, err := auditdb.LegalHolds.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, holdErrors.Map(err, req.ID.String())
		}
		return hold, nil
	})
}

func (r *repository) Create(ctx context.Context, hold *audit.LegalHold) (*audit.LegalHold, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.LegalHold, error) {
		if err := auditdb.LegalHolds.Create(ctx, r.db.Keel(ctx), hold); err != nil {
			return nil, holdErrors.Map(err, hold.ID.String())
		}
		return hold, nil
	})
}

func (r *repository) Release(ctx context.Context, hold *audit.LegalHold) (*audit.LegalHold, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.LegalHold, error) {
		if err := auditdb.LegalHolds.Save(ctx, r.db.Keel(ctx), hold); err != nil {
			return nil, holdErrors.Map(err, hold.ID.String())
		}
		return hold, nil
	})
}
