package scopepolicytest

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
)

type Policy struct {
	Registry          *permissioncatalog.Registry
	DatabaseEnforcing bool
}

func Enforcing() *Policy {
	return &Policy{Registry: permissioncatalog.NewRegistry(), DatabaseEnforcing: true}
}

func (p *Policy) Enforced(resource string, scope permission.DataScope) bool {
	return p.Registry.ScopeEnforced(resource, scope, p.DatabaseEnforcing)
}

func (p *Policy) EnforceableScopes(resource string) []permission.DataScope {
	return p.Registry.EnforceableScopes(resource, p.DatabaseEnforcing)
}
