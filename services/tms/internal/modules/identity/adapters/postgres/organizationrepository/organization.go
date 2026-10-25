package organizationrepository

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"unsafe"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	byIDsScopeReason         = "load organizations of the caller's business unit, including ones it shares no membership with"
	selectOptionsScopeReason = "list organizations of the caller's business unit to pick a counterparty from"
	loginSlugScopeReason     = "resolve an organization from its public login slug before any tenant is known"

	businessUnitScope = "business-unit"
)

var organizationErrors = dberror.Mapping{Entity: "Organization"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Cache  tenantports.OrganizationCacheRepository
}

type repository struct {
	db    *postgres.Connection
	cache tenantports.OrganizationCacheRepository
	l     *zap.Logger
}

func New(p Params) tenantports.OrganizationRepository {
	return &repository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.organization-repository"),
	}
}

func organizationLoads(includeState, includeBU bool) []keel.Loader {
	o := tenantdb.Organizations
	loads := make([]keel.Loader, 0, 2)
	if includeState {
		loads = append(loads, o.State())
	}
	if includeBU {
		loads = append(loads, o.BusinessUnit())
	}

	return loads
}

func (r *repository) GetByID(
	ctx context.Context,
	req tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.Organization, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.Any("request", req),
		)

		cachedOrg, err := r.cache.GetByID(ctx, req.TenantInfo.OrgID)
		if err == nil && cachedOrg.ID.IsNotNil() &&
			cachedOrg.BusinessUnitID == req.TenantInfo.BuID {
			needsRefresh := (req.IncludeState && cachedOrg.State == nil) ||
				(req.IncludeBU && cachedOrg.BusinessUnit == nil)
			if !needsRefresh {
				return cachedOrg, nil
			}

			log.Debug(
				"refreshing organization in cache",
				zap.String("orgID", cachedOrg.ID.String()),
			)
		}

		o := tenantdb.Organizations
		org, err := o.Where(
			o.ID.Eq(req.TenantInfo.OrgID),
			o.BusinessUnitID.Eq(req.TenantInfo.BuID),
		).
			Load(organizationLoads(req.IncludeState, req.IncludeBU)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			log.Error("failed to get organization from database", zap.Error(err))
			return nil, organizationErrors.Map(err, req.TenantInfo.OrgID.String())
		}

		return org, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req tenantports.GetOrganizationsByIDsQuery,
) ([]*tenant.Organization, error) {
	ctx = dbscope.WithSystem(ctx, byIDsScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.Organization, error) {
		if len(req.OrganizationIDs) == 0 {
			return []*tenant.Organization{}, nil
		}

		o := tenantdb.Organizations
		entities, err := o.AllTenants().
			Where(
				o.BusinessUnitID.Eq(req.TenantInfo.BuID),
				o.ID.In(req.OrganizationIDs...),
			).
			Load(organizationLoads(req.IncludeState, req.IncludeBU)...).
			Limit(len(req.OrganizationIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get organizations from database", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *tenantports.SelectOrganizationOptionsRequest,
) (*pagination.ListResult[*tenant.Organization], error) {
	ctx = dbscope.WithSystem(ctx, selectOptionsScopeReason)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tenant.Organization], error) {
			o := tenantdb.Organizations
			current := req.SelectQueryRequest.TenantInfo
			q := o.AllTenants().
				Pick(o.ID, o.BusinessUnitID, o.Name, o.ScacCode, o.City, o.StateID, o.LogoURL).
				Where(
					o.BusinessUnitID.Eq(current.BuID),
					keel.When(req.Scope != businessUnitScope, o.ID.Eq(current.OrgID)),
					keel.When(req.ExcludeCurrent, o.ID.Ne(current.OrgID)),
				).
				Order(o.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					search := strings.TrimSpace(s)
					if search == "" {
						return nil
					}
					return keel.Or(
						o.Name.ContainsFold(search),
						o.ScacCode.ContainsFold(search),
						o.City.ContainsFold(search),
					)
				},
			)
		},
	)
}

func (r *repository) GetByLoginSlug(
	ctx context.Context,
	loginSlug string,
) (*tenant.Organization, error) {
	ctx = dbscope.WithSystem(ctx, loginSlugScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.Organization, error) {
		o := tenantdb.Organizations
		org, err := o.AllTenants().Where(o.LoginSlug.Eq(loginSlug)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, organizationErrors.Map(err, loginSlug)
		}

		return org, nil
	})
}

func (r *repository) ListLoginSlugsByPrefix(
	ctx context.Context,
	prefix string,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		o := tenantdb.Organizations
		q := o.Pick(o.LoginSlug)
		if dbscope.IsSystem(ctx) {
			q = q.AllTenants()
		}

		return keel.Into[string](ctx, r.db.Keel(ctx), q.Where(keel.Or(
			o.LoginSlug.Eq(prefix),
			o.LoginSlug.HasPrefix(prefix+"-"),
		)))
	})
}

func (r *repository) Update(
	ctx context.Context,
	org *tenant.Organization,
) (*tenant.Organization, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.Organization, error) {
		o := tenantdb.Organizations
		db := r.db.Keel(ctx)

		stored, err := o.Where(o.ID.Eq(org.ID), o.BusinessUnitID.Eq(org.BusinessUnitID)).
			One(ctx, db)
		if err != nil {
			return nil, organizationErrors.Map(err, org.ID.String())
		}
		keepStoredWhereZero(org, stored, o.BrokerageEnabled, o.AssetOperationsEnabled)

		if err = o.Save(ctx, db, org); err != nil {
			r.l.Error(
				"failed to update organization in database",
				zap.String("orgID", org.ID.String()),
				zap.Error(err),
			)
			return nil, organizationErrors.Map(err, org.ID.String())
		}

		return org, nil
	})
}

func keepStoredWhereZero(org, stored *tenant.Organization, written ...keel.Expr) {
	table := tenantdb.OrganizationsSchema()
	always := make([]int32, 0, len(written)+2)
	always = append(always, table.Version, table.AutoUpdate)
	for _, e := range written {
		if _, index, ok := keel.ColumnOf(e); ok {
			always = append(always, index)
		}
	}

	dst, src := unsafe.Pointer(org), unsafe.Pointer(stored)
	for _, index := range table.Update {
		if slices.Contains(always, index) {
			continue
		}
		field := reflect.ValueOf(table.Ptr(dst, index)).Elem()
		if field.IsZero() {
			field.Set(reflect.ValueOf(table.Ptr(src, index)).Elem())
		}
	}
}

func (r *repository) ClearLogoURL(
	ctx context.Context,
	orgID pulid.ID,
	version int64,
) (*tenant.Organization, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.Organization, error) {
		o := tenantdb.Organizations
		rows, err := o.Where(o.ID.Eq(orgID), o.Version.Eq(version)).
			UpdateReturning(ctx, r.db.Keel(ctx), o.LogoURL.To(""))
		if err != nil {
			r.l.Error(
				"failed to clear organization logo URL",
				zap.String("orgID", orgID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckAffected(
			int64(len(rows)),
			"Organization",
			orgID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}
