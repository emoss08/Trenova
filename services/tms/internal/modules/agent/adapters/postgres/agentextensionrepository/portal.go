package agentextensionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal/extensionportaldb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	portalSiteEntityName = "Portal site"
	portalRunEntityName  = "Portal run"
	defaultPortalRuns    = 50
	maxPortalRuns        = 200
)

var (
	portalSiteErrors = dberror.Mapping{Entity: portalSiteEntityName}
	portalRunErrors  = dberror.Mapping{Entity: portalRunEntityName}
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
	_ pagination.TenantInfo,
) ([]*extensionportal.PortalSite, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*extensionportal.PortalSite, error) {
		s := extensionportaldb.PortalSites
		sites, err := s.Query().
			Order(s.Name.Asc()).
			Limit(extensionportal.MaxPortalSitesPerTenant).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list portal sites", zap.Error(err))
			return nil, fmt.Errorf("list portal sites: %w", err)
		}

		return sites, nil
	})
}

func (r *portalSiteRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*extensionportal.PortalSite, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalSite, error) {
		site, err := extensionportaldb.PortalSites.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, portalSiteErrors.Map(err, id.String())
		}

		return site, nil
	})
}

func (r *portalSiteRepository) Create(
	ctx context.Context,
	site *extensionportal.PortalSite,
) (*extensionportal.PortalSite, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalSite, error) {
		if err := extensionportaldb.PortalSites.Create(ctx, r.db.Keel(ctx), site); err != nil {
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
	site *extensionportal.PortalSite,
) (*extensionportal.PortalSite, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalSite, error) {
		if err := extensionportaldb.PortalSites.Save(ctx, r.db.Keel(ctx), site); err != nil {
			switch {
			case dberror.IsUniqueConstraintViolation(err):
				return nil, duplicateSiteName()
			case dberror.IsVersionConflict(err):
				return nil, portalSiteErrors.Map(err, site.ID.String())
			}
			r.l.Error("failed to update portal site", zap.Error(err))
			return nil, fmt.Errorf("update portal site: %w", err)
		}

		return site, nil
	})
}

func (r *portalSiteRepository) Delete(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := extensionportaldb.PortalSites
		n, err := s.Where(s.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsForeignKeyConstraintViolation(err) {
				return errortypes.NewBusinessError(
					"Agents have used this site, so its history is kept. Turn it off instead.",
				)
			}
			r.l.Error("failed to delete portal site", zap.Error(err))
			return fmt.Errorf("delete portal site: %w", err)
		}

		return dberror.CheckAffected(n, portalSiteEntityName, id.String())
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
	run *extensionportal.PortalRun,
) (*extensionportal.PortalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalRun, error) {
		if err := extensionportaldb.PortalRuns.Create(ctx, r.db.Keel(ctx), run); err != nil {
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
	_ pagination.TenantInfo,
	id pulid.ID,
) (*extensionportal.PortalRun, error) {
	return r.getWhere(ctx, extensionportaldb.PortalRuns.ID.Eq(id))
}

func (r *portalRunRepository) GetByIdempotencyKey(
	ctx context.Context,
	_ pagination.TenantInfo,
	key string,
) (*extensionportal.PortalRun, error) {
	return r.getWhere(ctx, extensionportaldb.PortalRuns.IdempotencyKey.Eq(key))
}

func (r *portalRunRepository) getWhere(
	ctx context.Context,
	where keel.Pred,
) (*extensionportal.PortalRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalRun, error) {
		run, err := extensionportaldb.PortalRuns.Where(where).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, portalRunErrors.Map(err, "")
		}

		return run, nil
	})
}

func (r *portalRunRepository) Update(
	ctx context.Context,
	run *extensionportal.PortalRun,
) (*extensionportal.PortalRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extensionportal.PortalRun, error) {
		if err := extensionportaldb.PortalRuns.Save(ctx, r.db.Keel(ctx), run); err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, portalRunErrors.Map(err, run.ID.String())
			}
			r.l.Error("failed to update portal run", zap.Error(err))
			return nil, fmt.Errorf("update portal run: %w", err)
		}

		return run, nil
	})
}

func (r *portalRunRepository) List(
	ctx context.Context,
	req *agentextensionports.ListPortalRunsRequest,
) ([]*extensionportal.PortalRun, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPortalRuns
	}
	limit = min(limit, maxPortalRuns)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*extensionportal.PortalRun, error) {
		p := extensionportaldb.PortalRuns
		runs, err := p.Where(
			keel.When(req.SiteID.IsNotNil(), p.SiteID.Eq(req.SiteID)),
			keel.When(req.Before > 0, p.StartedAt.Lt(req.Before)),
		).
			Order(p.StartedAt.Desc(), p.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list portal runs", zap.Error(err))
			return nil, fmt.Errorf("list portal runs: %w", err)
		}

		return runs, nil
	})
}
