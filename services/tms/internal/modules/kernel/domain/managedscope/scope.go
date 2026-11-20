package managedscope

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const InstanceKey = "instance"

var (
	ErrNoTenantScope = errors.New(
		"a plan question on a shared server needs the tenant it is about, and the context carries none",
	)
	ErrWrongTenancy = errors.New(
		"the scope does not belong to this server's tenancy: a dedicated server has only the instance scope, a shared server only organization scopes",
	)
)

type Scope struct {
	key    string
	tenant dbscope.Tenant
}

func Instance() Scope {
	return Scope{key: InstanceKey}
}

func Organization(tenant dbscope.Tenant) Scope {
	return Scope{
		key: tenant.OrganizationID.String(),
		tenant: dbscope.Tenant{
			OrganizationID: tenant.OrganizationID,
			BusinessUnitID: tenant.BusinessUnitID,
		},
	}
}

func FromKey(key string, businessUnitID pulid.ID) Scope {
	if key == InstanceKey {
		return Instance()
	}
	return Organization(dbscope.Tenant{
		OrganizationID: pulid.ID(key),
		BusinessUnitID: businessUnitID,
	})
}

func (s Scope) Key() string {
	return s.key
}

func (s Scope) IsZero() bool {
	return s.key == ""
}

func (s Scope) IsInstance() bool {
	return s.key == InstanceKey
}

func (s Scope) Tenant() (dbscope.Tenant, bool) {
	return s.tenant, !s.IsZero() && !s.IsInstance()
}

func (s Scope) BusinessUnitID() pulid.ID {
	return s.tenant.BusinessUnitID
}

type Resolver struct {
	shared bool
}

func NewResolver(shared bool) *Resolver {
	return &Resolver{shared: shared}
}

func (r *Resolver) Shared() bool {
	return r.shared
}

func (r *Resolver) Scope(ctx context.Context) (Scope, error) {
	if !r.shared {
		return Instance(), nil
	}
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || tenant.OrganizationID.IsNil() {
		return Scope{}, ErrNoTenantScope
	}
	return Organization(tenant), nil
}

func (r *Resolver) ForTenant(tenant dbscope.Tenant) (Scope, error) {
	if !r.shared {
		return Instance(), nil
	}
	if tenant.OrganizationID.IsNil() {
		return Scope{}, ErrNoTenantScope
	}
	return Organization(tenant), nil
}

func (r *Resolver) Accepts(scope Scope) error {
	if scope.IsZero() || scope.IsInstance() == r.shared {
		return ErrWrongTenancy
	}
	return nil
}
