package organizationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Cache  repositories.OrganizationCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache repositories.OrganizationCacheRepository
	l     *zap.Logger
}

func New(p Params) repositories.OrganizationRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.organization-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("request", req),
	)

	cachedOrg, err := r.cache.GetByID(ctx, req.TenantInfo.OrgID)
	if err == nil && cachedOrg.ID.IsNotNil() {
		log.Debug("organization found in cache", zap.String("orgID", cachedOrg.ID.String()))

		needsRefresh := (req.IncludeState && cachedOrg.State == nil) ||
			(req.IncludeBU && cachedOrg.BusinessUnit == nil)

		if !needsRefresh {
			return cachedOrg, nil
		}

		log.Debug("refreshing organization in cache", zap.String("orgID", cachedOrg.ID.String()))
	}

	org := new(tenant.Organization)
	q := r.db.DB().
		NewSelect().
		Model(org).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("org.id = ?", req.TenantInfo.OrgID).
				Where("org.business_unit_id = ?", req.TenantInfo.BuID)
		})

	if req.IncludeState {
		q.Relation("State")
	}

	if req.IncludeBU {
		q.Relation("BusinessUnit")
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get organization from database", zap.Error(err))
		return nil, err
	}

	return org, nil
}

func (r *repository) GetByLoginSlug(
	ctx context.Context,
	loginSlug string,
) (*tenant.Organization, error) {
	org := new(tenant.Organization)
	if err := r.db.DB().
		NewSelect().
		Model(org).
		Where("org.login_slug = ?", loginSlug).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Organization")
	}

	return org, nil
}

func (r *repository) ListLoginSlugsByPrefix(
	ctx context.Context,
	prefix string,
) ([]string, error) {
	var slugs []string
	if err := r.db.DB().
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Column("login_slug").
		Where("login_slug = ?", prefix).
		WhereOr("login_slug LIKE ?", prefix+"-%").
		Scan(ctx, &slugs); err != nil {
		return nil, err
	}

	return slugs, nil
}

func (r *repository) Update(
	ctx context.Context,
	org *tenant.Organization,
) (*tenant.Organization, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", org.ID.String()),
	)

	ov := org.Version
	org.Version++

	results, rErr := r.db.DB().
		NewUpdate().
		Model(org).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to update organization in database", zap.Error(rErr))
		return nil, rErr
	}

	if err := dberror.CheckRowsAffected(results, "Organization", org.ID.String()); err != nil {
		return nil, err
	}

	return org, nil
}

func (r *repository) ClearLogoURL(
	ctx context.Context,
	orgID pulid.ID,
	version int64,
) (*tenant.Organization, error) {
	log := r.l.With(
		zap.String("operation", "ClearLogoURL"),
		zap.String("orgID", orgID.String()),
	)

	org := &tenant.Organization{
		ID:      orgID,
		Version: version + 1,
	}

	results, rErr := r.db.DB().
		NewUpdate().
		Model(org).
		Set("logo_url = ''").
		Set("version = ?", org.Version).
		Where("org.id = ?", orgID).
		Where("org.version = ?", version).
		Returning("*").
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to clear organization logo URL", zap.Error(rErr))
		return nil, rErr
	}

	if err := dberror.CheckRowsAffected(results, "Organization", orgID.String()); err != nil {
		return nil, err
	}

	return org, nil
}
