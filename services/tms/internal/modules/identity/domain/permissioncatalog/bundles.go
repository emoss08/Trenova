package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type BundleGrant struct {
	Resource   permission.Resource    `json:"resource"`
	Operations []permission.Operation `json:"operations"`
	Scope      permission.DataScope   `json:"scope,omitempty"`
}

type Bundle struct {
	ID          string        `json:"id"`
	Module      string        `json:"module"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Grants      []BundleGrant `json:"grants"`
}

type JobTemplate struct {
	ID             string                      `json:"id"`
	Label          string                      `json:"label"`
	Description    string                      `json:"description"`
	Bundles        []string                    `json:"bundles"`
	MaxSensitivity permission.FieldSensitivity `json:"maxSensitivity"`
	ReadEverything bool                        `json:"readEverything,omitempty"`
	ReadExcept     []permission.Resource       `json:"readExcept,omitempty"`
}

func Bundles() []Bundle {
	return slices.Clone(capabilityBundles)
}

func JobTemplates() []JobTemplate {
	return slices.Clone(jobTemplates)
}

func BundleByID(id string) (Bundle, bool) {
	idx := slices.IndexFunc(capabilityBundles, func(b Bundle) bool { return b.ID == id })
	if idx < 0 {
		return Bundle{}, false
	}
	return capabilityBundles[idx], true
}

func (r *Registry) TemplateGrants(template JobTemplate) []*permission.ResourcePermission {
	byResource := make(map[string]*permission.ResourcePermission)
	order := make([]string, 0, 32)
	add := func(resource string, ops []permission.Operation, scope permission.DataScope) {
		if scope == "" {
			scope = permission.DataScopeOrganization
		}
		existing, ok := byResource[resource]
		if !ok {
			byResource[resource] = &permission.ResourcePermission{
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
	grants := make([]*permission.ResourcePermission, 0, len(order))
	for _, resource := range order {
		grants = append(grants, byResource[resource])
	}
	return grants
}
