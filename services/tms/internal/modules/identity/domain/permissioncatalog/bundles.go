package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dispatchbundles"
	"github.com/emoss08/trenova/internal/modules/identity/domain/financebundles"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformbundles"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/workforcebundles"
)

var capabilityBundles = slices.Concat(
	dispatchbundles.All(),
	financebundles.All(),
	workforcebundles.All(),
	platformbundles.All(),
)

func Bundles() []permissionspec.Bundle {
	return slices.Clone(capabilityBundles)
}

func JobTemplates() []permissionspec.JobTemplate {
	return slices.Clone(jobTemplates)
}

func BundleByID(id string) (permissionspec.Bundle, bool) {
	idx := slices.IndexFunc(
		capabilityBundles,
		func(b permissionspec.Bundle) bool { return b.ID == id },
	)
	if idx < 0 {
		return permissionspec.Bundle{}, false
	}
	return capabilityBundles[idx], true
}

func (r *Registry) TemplateGrants(
	template permissionspec.JobTemplate,
) []*rbac.ResourcePermission {
	byResource := make(map[string]*rbac.ResourcePermission)
	order := make([]string, 0, 32)
	add := func(resource string, ops []permission.Operation, scope permission.DataScope) {
		if scope == "" {
			scope = permission.DataScopeOrganization
		}
		existing, ok := byResource[resource]
		if !ok {
			byResource[resource] = &rbac.ResourcePermission{
				Resource:   resource,
				Operations: permission.Normalize(ops),
				DataScope:  scope,
			}
			order = append(order, resource)
			return
		}
		existing.Operations = permission.Normalize(append(existing.Operations, ops...))
		if scope.IsMorePermissive(existing.DataScope) {
			existing.DataScope = scope
		}
	}

	if template.ReadEverything {
		for _, def := range r.All() {
			if slices.Contains(template.ReadExcept, permission.Resource(def.Resource)) {
				continue
			}
			if slices.Contains(r.GetOperationsForResource(def.Resource), permission.OpRead) {
				add(
					def.Resource,
					[]permission.Operation{permission.OpRead},
					permission.DataScopeOrganization,
				)
			}
		}
	}
	for _, id := range template.Bundles {
		bundle, ok := BundleByID(id)
		if !ok {
			continue
		}
		for _, grant := range bundle.Grants {
			add(grant.Resource.String(), grant.Operations, grant.Scope)
		}
	}

	slices.Sort(order)
	grants := make([]*rbac.ResourcePermission, 0, len(order))
	for _, resource := range order {
		grants = append(grants, byResource[resource])
	}
	return grants
}
