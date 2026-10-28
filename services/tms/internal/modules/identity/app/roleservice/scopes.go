package roleservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func unenforcedScopes(
	policy scopePolicy,
	registry *permissioncatalog.Registry,
	permissions []*permission.ResourcePermission,
	existing []*permission.ResourcePermission,
	field func(index int) string,
) *errortypes.MultiError {
	kept := make(map[string]permission.DataScope, len(existing))
	for _, rp := range existing {
		if rp != nil {
			kept[rp.Resource] = rp.DataScope
		}
	}

	me := errortypes.NewMultiError()
	for i, rp := range permissions {
		if rp == nil || !rp.DataScope.IsNarrow() {
			continue
		}
		if scope, ok := kept[rp.Resource]; ok && scope == rp.DataScope {
			continue
		}
		if policy.Enforced(rp.Resource, rp.DataScope) {
			continue
		}
		name := rp.Resource
		if def, ok := registry.Get(rp.Resource); ok {
			name = def.DisplayName
		}
		if rp.DataScope == permission.DataScopeTeam {
			me.Add(
				field(i),
				errortypes.ErrInvalid,
				"Team-only access to {0} is not enforced here, so it would reach the whole organization; choose Organization",
				name,
			)
			continue
		}
		me.Add(
			field(i),
			errortypes.ErrInvalid,
			"Own-records-only access to {0} is not enforced here, so it would reach the whole organization; choose Organization",
			name,
		)
	}
	if !me.HasErrors() {
		return nil
	}
	return me
}

func permissionField(index int) string {
	return fmt.Sprintf("permissions[%d].dataScope", index)
}

func singleField(int) string {
	return "dataScope"
}
