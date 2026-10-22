package permission

import (
	"slices"
)

type BundleGrant struct {
	Resource   Resource    `json:"resource"`
	Operations []Operation `json:"operations"`
	Scope      DataScope   `json:"scope,omitempty"`
}

type Bundle struct {
	ID          string        `json:"id"`
	Module      string        `json:"module"`
	Label       string        `json:"label"`
	Description string        `json:"description"`
	Grants      []BundleGrant `json:"grants"`
}

type JobTemplate struct {
	ID             string           `json:"id"`
	Label          string           `json:"label"`
	Description    string           `json:"description"`
	Bundles        []string         `json:"bundles"`
	MaxSensitivity FieldSensitivity `json:"maxSensitivity"`
	ReadEverything bool             `json:"readEverything,omitempty"`
	ReadExcept     []Resource       `json:"readExcept,omitempty"`
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

func (r *Registry) TemplateGrants(template JobTemplate) []*ResourcePermission {
	byResource := make(map[string]*ResourcePermission)
	order := make([]string, 0, 32)
	add := func(resource string, ops []Operation, scope DataScope) {
		if scope == "" {
			scope = DataScopeOrganization
		}
		existing, ok := byResource[resource]
		if !ok {
			byResource[resource] = &ResourcePermission{
				Resource:   resource,
				Operations: Normalize(ops),
				DataScope:  scope,
			}
			order = append(order, resource)
			return
		}
		existing.Operations = Normalize(append(existing.Operations, ops...))
		if scope.IsMorePermissive(existing.DataScope) {
			existing.DataScope = scope
		}
	}

	if template.ReadEverything {
		for _, def := range r.All() {
			if slices.Contains(template.ReadExcept, Resource(def.Resource)) {
				continue
			}
			if slices.Contains(r.GetOperationsForResource(def.Resource), OpRead) {
				add(def.Resource, []Operation{OpRead}, DataScopeOrganization)
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
	grants := make([]*ResourcePermission, 0, len(order))
	for _, resource := range order {
		grants = append(grants, byResource[resource])
	}
	return grants
}
