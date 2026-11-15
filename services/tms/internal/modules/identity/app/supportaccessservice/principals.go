package supportaccessservice

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
)

// principalTTL bounds how long a process serves a support principal's permissions without
// reading its grant again. A revoked grant's sessions are deleted at once and every request
// reads the grant (RequireLive), so this only bounds work already past sign-in.
const (
	principalTTL       = 5 * time.Second
	principalCacheSize = 1_000
)

type PrincipalsParams struct {
	fx.In

	Grants   supportaccessports.GrantRepository
	Registry *permissioncatalog.Registry
	Managed  *managedconfig.Config
}

// Principals is the support principal's side of the permission engine and of sign-in. It
// depends on nothing that depends on the engine.
type Principals struct {
	grants   supportaccessports.GrantRepository
	registry *permissioncatalog.Registry
	enabled  bool
	cache    *otter.Cache[pulid.ID, principalGrant]

	once     sync.Once
	readOnly map[string]*permissionports.CachedResourcePermission
	full     map[string]*permissionports.CachedResourcePermission
}

type principalGrant struct {
	grant *supportaccess.Grant
}

var _ supportaccessports.Principals = (*Principals)(nil)

func NewPrincipals(p PrincipalsParams) *Principals {
	return &Principals{
		grants:   p.Grants,
		registry: p.Registry,
		enabled:  p.Managed.Enabled,
		cache: otter.Must(&otter.Options[pulid.ID, principalGrant]{
			MaximumSize:      principalCacheSize,
			ExpiryCalculator: otter.ExpiryWriting[pulid.ID, principalGrant](principalTTL),
		}),
	}
}

// Permissions answers for a support principal what a grant of its scope allows, and nothing
// when its grant is not live, is for another organization, or support access is off.
func (p *Principals) Permissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*permissionports.CachedPermissions, error) {
	now := timeutils.NowUnix()
	empty := &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityPublic),
		Resources:      map[string]*permissionports.CachedResourcePermission{},
		ExpiresAt:      now,
	}
	if !p.enabled || !supportaccess.IsPrincipalID(userID) {
		return empty, nil
	}

	grant, err := p.grantOf(ctx, userID, orgID)
	if errors.Is(err, supportaccess.ErrGrantNotFound) {
		return empty, nil
	}
	if err != nil {
		return nil, err
	}
	if !grant.LiveAt(now) || grant.OrganizationID != orgID || grant.PrincipalUserID != userID {
		return empty, nil
	}

	return &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityConfidential),
		Resources:      p.resourcesFor(grant.Scope),
		ExpiresAt:      grant.ExpiresAt,
	}, nil
}

// RequireLive reads the grant of a support session, uncached, and returns
// supportaccess.ErrGrantNotFound unless it is live and bound to that session's principal.
func (p *Principals) RequireLive(ctx context.Context, req *supportaccessports.RequireLiveRequest) error {
	if !p.enabled {
		return supportaccess.ErrGrantNotFound
	}
	grant, err := p.grants.ByPrincipal(
		dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
			UserID:         req.PrincipalUserID,
		}),
		pagination.TenantInfo{
			OrgID:  req.OrganizationID,
			BuID:   req.BusinessUnitID,
			UserID: req.PrincipalUserID,
		},
		req.PrincipalUserID,
	)
	if err != nil {
		return err
	}
	if grant.ID != req.GrantID || !grant.LiveAt(req.Now) {
		p.cache.Invalidate(req.PrincipalUserID)
		return supportaccess.ErrGrantNotFound
	}
	p.cache.Set(req.PrincipalUserID, principalGrant{grant: grant})
	return nil
}

// Forget drops what this process holds about a principal, after a revocation.
func (p *Principals) Forget(userID pulid.ID) {
	p.cache.Invalidate(userID)
}

func (p *Principals) grantOf(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*supportaccess.Grant, error) {
	if cached, ok := p.cache.GetIfPresent(userID); ok {
		return cached.grant, nil
	}

	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || tenant.OrganizationID != orgID {
		return nil, supportaccess.ErrGrantNotFound
	}
	grant, err := p.grants.ByPrincipal(ctx, pagination.TenantInfo{
		OrgID:  tenant.OrganizationID,
		BuID:   tenant.BusinessUnitID,
		UserID: userID,
	}, userID)
	if err != nil {
		return nil, err
	}
	p.cache.Set(userID, principalGrant{grant: grant})
	return grant, nil
}

func (p *Principals) resourcesFor(scope supportaccess.Scope) map[string]*permissionports.CachedResourcePermission {
	p.once.Do(p.build)
	switch scope {
	case supportaccess.ScopeFull:
		return maps.Clone(p.full)
	case supportaccess.ScopeReadOnly:
		return maps.Clone(p.readOnly)
	default:
		return map[string]*permissionports.CachedResourcePermission{}
	}
}

func (p *Principals) build() {
	p.readOnly = SupportPermissions(p.registry, supportaccess.ScopeReadOnly)
	p.full = SupportPermissions(p.registry, supportaccess.ScopeFull)
}

// SupportPermissions is what a grant of scope holds, organization-wide, as the engine reads it
// (supportaccess.Effective over the registry).
func SupportPermissions(
	registry *permissioncatalog.Registry,
	scope supportaccess.Scope,
) map[string]*permissionports.CachedResourcePermission {
	effective := supportaccess.Effective(scope, registry.All())
	out := make(map[string]*permissionports.CachedResourcePermission, len(effective))
	for resource, ops := range effective {
		held := make([]string, len(ops))
		for i, op := range ops {
			held[i] = string(op)
		}
		out[resource] = &permissionports.CachedResourcePermission{
			Operations: held,
			DataScope:  string(permission.DataScopeOrganization),
		}
	}
	return out
}
