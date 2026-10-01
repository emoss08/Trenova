package dbscope

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

type Kind uint8

const (
	KindNone Kind = iota
	KindTenant
	KindSystem
)

func (k Kind) String() string {
	switch k {
	case KindTenant:
		return "tenant"
	case KindSystem:
		return "system"
	default:
		return "none"
	}
}

type Tenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
}

func (t Tenant) Valid() bool {
	return !t.OrganizationID.IsNil() && !t.BusinessUnitID.IsNil()
}

type Scope struct {
	kind   Kind
	tenant Tenant
	reason string
}

func (s Scope) Kind() Kind {
	return s.kind
}

func (s Scope) Tenant() (Tenant, bool) {
	return s.tenant, s.kind == KindTenant
}

func (s Scope) Reason() string {
	return s.reason
}

func (s Scope) Matches(other Scope) bool {
	return s == other
}

type scopeKey struct{}

func WithTenant(ctx context.Context, tenant Tenant) context.Context {
	return context.WithValue(ctx, scopeKey{}, Scope{kind: KindTenant, tenant: tenant})
}

func WithSystem(ctx context.Context, reason string) context.Context {
	return context.WithValue(
		ctx,
		scopeKey{},
		Scope{kind: KindSystem, reason: strings.TrimSpace(reason)},
	)
}

func From(ctx context.Context) Scope {
	if ctx == nil {
		return Scope{}
	}

	scope, _ := ctx.Value(scopeKey{}).(Scope)

	return scope
}

func TenantFrom(ctx context.Context) (Tenant, bool) {
	return From(ctx).Tenant()
}

func IsSystem(ctx context.Context) bool {
	return From(ctx).kind == KindSystem
}
