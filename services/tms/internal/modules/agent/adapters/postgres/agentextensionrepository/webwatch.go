package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentextensioncol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	webWatchEntityName   = "Web watch"
	defaultWebWatches    = 50
	maxWebWatches        = 200
	defaultWatchFindings = 20
	maxWatchFindings     = 100
	maxUndelivered       = 50
)

type webWatchRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWebWatchRepository(p Params) agentextensionports.WebWatchRepository {
	return &webWatchRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-web-watch-repository"),
	}
}

func (r *webWatchRepository) Create(
	ctx context.Context,
	watch *agentextension.WebWatch,
) (*agentextension.WebWatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(watch).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to create web watch", zap.Error(err))
			return nil, fmt.Errorf("create web watch: %w", err)
		}

		return watch, nil
	})
}

func (r *webWatchRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.WebWatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		watch := new(agentextension.WebWatch)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(watch).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentextensioncol.WebWatchScopeTenant(sq, tenantInfo).
					Where(agentextensioncol.WebWatchColumns.ID.Eq(), id)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, webWatchEntityName)
		}

		return watch, nil
	})
}

func (r *webWatchRepository) Update(
	ctx context.Context,
	watch *agentextension.WebWatch,
) (*agentextension.WebWatch, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.WebWatch, error) {
		ov := watch.Version
		watch.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(watch).
			WherePK().
			Where(agentextensioncol.WebWatchColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			watch.Version = ov
			r.l.Error("failed to update web watch", zap.Error(err))
			return nil, fmt.Errorf("update web watch: %w", err)
		}
		if err = dberror.CheckRowsAffected(result, webWatchEntityName, watch.ID.String()); err != nil {
			watch.Version = ov
			return nil, err
		}

		return watch, nil
	})
}

func (r *webWatchRepository) List(
	ctx context.Context,
	req *agentextensionports.ListWebWatchesRequest,
) ([]*agentextension.WebWatch, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultWebWatches
	}
	limit = min(limit, maxWebWatches)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.WebWatch, error) {
		watches := make([]*agentextension.WebWatch, 0, limit)
		cols := agentextensioncol.WebWatchColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&watches).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = agentextensioncol.WebWatchScopeTenant(sq, req.TenantInfo)
				if req.Status != "" {
					sq = sq.Where(cols.Status.Eq(), req.Status)
				}
				if req.ThreadID.IsNotNil() {
					sq = sq.Where(cols.ThreadID.Eq(), req.ThreadID)
				}
				if req.RequestedBy.IsNotNil() {
					sq = sq.Where(cols.RequestedByID.Eq(), req.RequestedBy)
				}
				if req.DefinitionID.IsNotNil() {
					sq = sq.Where(cols.AgentDefinitionID.Eq(), req.DefinitionID)
				}
				if req.Before > 0 {
					sq = sq.Where(cols.CreatedAt.Lt(), req.Before)
				}
				return sq
			}).
			Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list web watches", zap.Error(err))
			return nil, fmt.Errorf("list web watches: %w", err)
		}

		return watches, nil
	})
}

func (r *webWatchRepository) CountActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agentextension.WebWatch)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentextensioncol.WebWatchScopeTenant(sq, tenantInfo).
					Where(agentextensioncol.WebWatchColumns.Status.Eq(),
						agentextension.WebWatchStatusActive)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count active web watches", zap.Error(err))
			return 0, fmt.Errorf("count active web watches: %w", err)
		}

		return count, nil
	})
}

type webWatchFindingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewWebWatchFindingRepository(p Params) agentextensionports.WebWatchFindingRepository {
	return &webWatchFindingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-web-watch-finding-repository"),
	}
}

func (r *webWatchFindingRepository) Create(
	ctx context.Context,
	finding *agentextension.WebWatchFinding,
) (*agentextension.WebWatchFinding, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*agentextension.WebWatchFinding, error) {
			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(finding).
				Returning("*").
				Exec(ctx); err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, agentextensionports.ErrWebWatchFindingExists
				}
				r.l.Error("failed to record a web watch finding", zap.Error(err))
				return nil, fmt.Errorf("create web watch finding: %w", err)
			}

			return finding, nil
		},
	)
}

func (r *webWatchFindingRepository) ListByWatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watchID pulid.ID,
	limit int,
) ([]*agentextension.WebWatchFinding, error) {
	if limit <= 0 {
		limit = defaultWatchFindings
	}
	limit = min(limit, maxWatchFindings)

	return r.list(ctx, &findingQuery{
		tenant:  tenantInfo,
		watchID: watchID,
		limit:   limit,
		newest:  true,
	})
}

func (r *webWatchFindingRepository) ListUndelivered(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	watchID pulid.ID,
) ([]*agentextension.WebWatchFinding, error) {
	return r.list(ctx, &findingQuery{
		tenant:      tenantInfo,
		watchID:     watchID,
		limit:       maxUndelivered,
		undelivered: true,
	})
}

type findingQuery struct {
	tenant      pagination.TenantInfo
	watchID     pulid.ID
	limit       int
	undelivered bool
	newest      bool
}

func (r *webWatchFindingRepository) list(
	ctx context.Context,
	q *findingQuery,
) ([]*agentextension.WebWatchFinding, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*agentextension.WebWatchFinding, error) {
			findings := make([]*agentextension.WebWatchFinding, 0, q.limit)
			cols := agentextensioncol.WebWatchFindingColumns
			order := cols.FoundAt.OrderAsc()
			if q.newest {
				order = cols.FoundAt.OrderDesc()
			}
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&findings).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = agentextensioncol.WebWatchFindingScopeTenant(sq, q.tenant).
						Where(cols.WatchID.Eq(), q.watchID)
					if q.undelivered {
						sq = sq.Where(cols.DeliveredAt.IsNull())
					}
					return sq
				}).
				Order(order, cols.ID.OrderAsc()).
				Limit(int64(q.limit)).
				Scan(ctx)
			if err != nil {
				r.l.Error("failed to list web watch findings", zap.Error(err))
				return nil, fmt.Errorf("list web watch findings: %w", err)
			}

			return findings, nil
		},
	)
}

func (r *webWatchFindingRepository) MarkDelivered(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
	at int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := agentextensioncol.WebWatchFindingColumns
		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agentextension.WebWatchFinding)(nil)).
			Set(cols.DeliveredAt.Set(), at).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return agentextensioncol.WebWatchFindingScopeTenantUpdate(uq, tenantInfo).
					Where(cols.ID.In(), bun.List(ids)).
					Where(cols.DeliveredAt.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to mark web watch findings delivered", zap.Error(err))
			return fmt.Errorf("mark web watch findings delivered: %w", err)
		}

		return nil
	})
}
