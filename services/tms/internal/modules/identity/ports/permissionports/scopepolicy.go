package permissionports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/readscope"
)

type ScopePolicy interface {
	Enforced(resource string, scope permission.DataScope) bool
	EnforceableScopes(resource string) []permission.DataScope
}

type ReadScopes interface {
	Resolver(principal *PermissionCheckRequest) readscope.Resolver
}
