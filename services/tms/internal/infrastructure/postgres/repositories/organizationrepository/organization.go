package organizationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func NewRepository(p Params) repositories.OrganizationRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.organization-repository"),
	}
}

func (or *repository) filterQuery(
	q *bun.SelectQuery,
	f *pagination.QueryOptions,
) *bun.SelectQuery {
	return q.Where("org.business_unit_id = ?", f.TenantOpts.BuID).
		Limit(f.Limit).
		Offset(f.Offset)
}

func (or *repository) GetByID(
	ctx context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	log := or.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("request", req),
	)

	db, err := or.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	cachedOrg, err := or.cache.GetByID(ctx, req.OrgID)
	if err == nil && cachedOrg.ID.IsNotNil() {
		log.Debug("retrieved organization from cache", zap.String("orgID", req.OrgID.String()))

		needsRefresh := (req.IncludeState && cachedOrg.State == nil) ||
			(req.IncludeBu && cachedOrg.BusinessUnit == nil)

		if !needsRefresh {
			return cachedOrg, nil
		}

		log.Debug("cached organization missing requested relationships, refreshing from database")
	}

	org := new(tenant.Organization)
	q := db.NewSelect().
		Model(org).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("org.id = ?", req.OrgID).Where("org.business_unit_id = ?", req.BuID)
		})

	if req.IncludeState {
		q.Relation("State")
	}

	if req.IncludeBu {
		q.Relation("BusinessUnit")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "organization")
	}

	if err = or.cache.Set(ctx, org); err != nil {
		log.Error("failed to set organization in cache", zap.Error(err))
	}

	return org, nil
}

func (or *repository) GetUserOrganizations(
	ctx context.Context,
	opts *pagination.QueryOptions,
) (*pagination.ListResult[*tenant.Organization], error) {
	log := or.l.With(
		zap.String("operation", "GetUserOrganizations"),
		zap.Any("tenantOpts", opts.TenantOpts),
	)

	db, err := or.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	orgs, err := or.cache.GetUserOrganizations(ctx, opts.TenantOpts.UserID)

	if err == nil && len(orgs) > 0 {
		log.Debug("got user organizations from cache", zap.Int("count", len(orgs)))
		return &pagination.ListResult[*tenant.Organization]{
			Items: orgs,
			Total: len(orgs),
		}, nil
	}

	dbOrgs := make([]*tenant.Organization, 0, opts.Limit)

	q := db.NewSelect().
		Model(&dbOrgs).
		Relation("State").
		Join("INNER JOIN user_organization_memberships AS uom ON uom.organization_id = org.id").
		Where("uom.user_id = ?", opts.TenantOpts.UserID)

	q = or.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error(
			"failed to scan user organizations",
			zap.Error(err),
		)
		return nil, err
	}

	if err = or.cache.SetUserOrganizations(ctx, opts.TenantOpts.UserID, dbOrgs); err != nil {
		log.Error(
			"failed to set user organizations in cache",
			zap.Error(err),
		)
		// ! Do not return the error because it will not affect the user experience
	}

	return &pagination.ListResult[*tenant.Organization]{
		Items: dbOrgs,
		Total: total,
	}, nil
}

func (or *repository) Update(
	ctx context.Context,
	org *tenant.Organization,
) (*tenant.Organization, error) {
	log := or.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", org.ID.String()),
		zap.Int64("version", org.Version),
	)

	dba, err := or.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	ov := org.Version
	org.Version++

	results, rErr := dba.NewUpdate().Model(org).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update organization", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Organization", org.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	if err = or.cache.InvalidateOrganizationForAllUsers(ctx, org.ID); err != nil {
		log.Error("failed to invalidate organization cache for all users", zap.Error(err))
		// ! Do not return the error because it will not affect the user experience
	}

	return org, nil
}
