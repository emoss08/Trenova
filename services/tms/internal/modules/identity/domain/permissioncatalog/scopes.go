package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

func (r *Registry) scopeSupport(
	resource string,
	scope permission.DataScope,
) (permissionspec.ScopeSupport, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return permissionspec.ScopeSupport{}, false
	}
	idx := slices.IndexFunc(
		def.Scopes,
		func(s permissionspec.ScopeSupport) bool { return s.Scope == scope },
	)
	if idx < 0 {
		return permissionspec.ScopeSupport{}, false
	}
	return def.Scopes[idx], true
}

func (r *Registry) ScopeEnforced(
	resource string,
	scope permission.DataScope,
	databaseEnforcing bool,
) bool {
	if !scope.IsNarrow() {
		return scope.IsValid()
	}
	support, ok := r.scopeSupport(resource, scope)
	if !ok {
		return false
	}
	return support.EnforcedBy != permissionspec.ScopeEnforcedByDatabase || databaseEnforcing
}

func (r *Registry) ScopeDeclared(resource string, scope permission.DataScope) bool {
	if !scope.IsNarrow() {
		return scope.IsValid()
	}
	_, ok := r.scopeSupport(resource, scope)
	return ok
}

func (r *Registry) EnforcedByDatabase(resource string, scope permission.DataScope) bool {
	support, ok := r.scopeSupport(resource, scope)
	return ok && support.EnforcedBy == permissionspec.ScopeEnforcedByDatabase
}

func (r *Registry) EnforceableScopes(
	resource string,
	databaseEnforcing bool,
) []permission.DataScope {
	scopes := make([]permission.DataScope, 0, 3)
	for _, scope := range []permission.DataScope{permission.DataScopeOwn, permission.DataScopeTeam} {
		if r.ScopeEnforced(resource, scope, databaseEnforcing) {
			scopes = append(scopes, scope)
		}
	}
	return append(scopes, permission.DataScopeOrganization)
}

func (r *Registry) DatabaseScopedResources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]string, 0, 4)
	for name, def := range r.resources {
		if slices.ContainsFunc(def.Scopes, func(s permissionspec.ScopeSupport) bool {
			return s.EnforcedBy == permissionspec.ScopeEnforcedByDatabase
		}) {
			resources = append(resources, name)
		}
	}
	slices.Sort(resources)
	return resources
}

func (r *Registry) ResourcesDeclaring(scope permission.DataScope) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resources := make([]string, 0, 8)
	for name, def := range r.resources {
		if slices.ContainsFunc(
			def.Scopes,
			func(s permissionspec.ScopeSupport) bool { return s.Scope == scope },
		) {
			resources = append(resources, name)
		}
	}
	slices.Sort(resources)
	return resources
}
