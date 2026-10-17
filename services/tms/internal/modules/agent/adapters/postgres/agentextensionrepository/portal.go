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
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	portalSiteEntityName = "Portal site"
	portalRunEntityName  = "Portal run"
	defaultPortalRuns    = 50
	maxPortalRuns        = 200
)

type portalSiteRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPortalSiteRepository(p Params) agentextensionports.PortalSiteRepository {
	return &portalSiteRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-portal-site-repository"),
	}
}

func (r *portalSiteRepository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*agentextension.PortalSite, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.PortalSite, error) {
		sites := make([]*agentextension.PortalSite, 0, 8)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&sites).
			Apply(agentextensioncol.PortalSiteApplyTenant(tenantInfo)).
			Order(agentextensioncol.PortalSiteColumns.Name.OrderAsc()).
			Limit(agentextension.MaxPortalSitesPerTenant).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list portal sites", zap.Error(err))
			return nil, fmt.Errorf("list portal sites: %w", err)
		}

		return sites, nil
	})
}

func (r *portalSiteRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.PortalSite, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.PortalSite, error) {
		site := new(agentextension.PortalSite)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(site).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentextensioncol.PortalSiteScopeTenant(sq, tenantInfo).
					Where(agentextensioncol.PortalSiteColumns.ID.Eq(), id)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, portalSiteEntityName)
		}

		return site, nil
	})
}

func (r *portalSiteRepository) Create(
	ctx context.Context,
	site *agentextension.PortalSite,
) (*agentextension.PortalSite, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.PortalSite, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(site).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateSiteName()
			}
			r.l.Error("failed to create portal site", zap.Error(err))
			return nil, fmt.Errorf("create portal site: %w", err)
		}

		return site, nil
	})
}

func (r *portalSiteRepository) Update(
	ctx context.Context,
	site *agentextension.PortalSite,
) (*agentextension.PortalSite, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.PortalSite, error) {
		ov := site.Version
		site.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(site).
			WherePK().
			Where(agentextensioncol.PortalSiteColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateSiteName()
			}
			r.l.Error("failed to update portal site", zap.Error(err))
			return nil, fmt.Errorf("update portal site: %w", err)
		}
		if err = dberror.CheckRowsAffected(result, portalSiteEntityName, site.ID.String()); err != nil {
			return nil, err
		}

		return site, nil
	})
}

func (r *portalSiteRepository) Delete(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*agentextension.PortalSite)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return agentextensioncol.PortalSiteScopeTenantDelete(dq, tenantInfo).
					Where(agentextensioncol.PortalSiteColumns.ID.Eq(), id)
			}).
			Exec(ctx)
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return errortypes.NewBusinessError(
					"Agents have used this site, so its history is kept. Turn it off instead.",
				)
			}
			r.l.Error("failed to delete portal site", zap.Error(err))
			return fmt.Errorf("delete portal site: %w", err)
		}

		return dberror.CheckRowsAffected(result, portalSiteEntityName, id.String())
	})
}

func duplicateSiteName() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("name", errortypes.ErrDuplicate, "Another portal site already has this name")

	return multiErr
}

type portalRunRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPortalRunRepository(p Params) agentextensionports.PortalRunRepository {
	return &portalRunRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-portal-run-repository"),
	}
}

func (r *portalRunRepository) Create(
	ctx context.Context,
	run *agentextension.PortalRun,
) (*agentextension.PortalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.PortalRun, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(run).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, agentextensionports.ErrPortalRunExists
			}
			r.l.Error("failed to create portal run", zap.Error(err))
			return nil, fmt.Errorf("create portal run: %w", err)
		}

		return run, nil
	})
}

func (r *portalRunRepository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*agentextension.PortalRun, error) {
	return r.getWhere(ctx, tenantInfo, func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(agentextensioncol.PortalRunColumns.ID.Eq(), id)
	})
}

func (r *portalRunRepository) GetByIdempotencyKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
) (*agentextension.PortalRun, error) {
	return r.getWhere(ctx, tenantInfo, func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(agentextensioncol.PortalRunColumns.IdempotencyKey.Eq(), key)
	})
}

func (r *portalRunRepository) getWhere(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	where func(*bun.SelectQuery) *bun.SelectQuery,
) (*agentextension.PortalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentextension.PortalRun, error) {
		run := new(agentextension.PortalRun)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(run).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return where(agentextensioncol.PortalRunScopeTenant(sq, tenantInfo))
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, portalRunEntityName)
		}

		return run, nil
	})
}

func (r *portalRunRepository) Update(
	ctx context.Context,
	run *agentextension.PortalRun,
) (*agentextension.PortalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentextension.PortalRun, error) {
		ov := run.Version
		run.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(run).
			WherePK().
			Where(agentextensioncol.PortalRunColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update portal run", zap.Error(err))
			return nil, fmt.Errorf("update portal run: %w", err)
		}
		if err = dberror.CheckRowsAffected(result, portalRunEntityName, run.ID.String()); err != nil {
			return nil, err
		}

		return run, nil
	})
}

func (r *portalRunRepository) List(
	ctx context.Context,
	req *agentextensionports.ListPortalRunsRequest,
) ([]*agentextension.PortalRun, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPortalRuns
	}
	limit = min(limit, maxPortalRuns)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentextension.PortalRun, error) {
		runs := make([]*agentextension.PortalRun, 0, limit)
		cols := agentextensioncol.PortalRunColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&runs).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = agentextensioncol.PortalRunScopeTenant(sq, req.TenantInfo)
				if req.SiteID.IsNotNil() {
					sq = sq.Where(cols.SiteID.Eq(), req.SiteID)
				}
				if req.Before > 0 {
					sq = sq.Where(cols.StartedAt.Lt(), req.Before)
				}
				return sq
			}).
			Order(cols.StartedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list portal runs", zap.Error(err))
			return nil, fmt.Errorf("list portal runs: %w", err)
		}

		return runs, nil
	})
}
