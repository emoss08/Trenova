package permissionresolver

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/identity/app/apikeyservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

func operationStrings(ops []permission.Operation) []string {
	out := make([]string, len(ops))
	for i, op := range ops {
		out[i] = string(op)
	}
	return out
}

type textFunc func(string) string

func translator(ctx context.Context) textFunc {
	return func(text string) string { return i18n.T(ctx, text) }
}

func verbatim(text string) string { return text }

func operationModels(
	text textFunc,
	defs []permissionspec.OperationDefinition,
) []*gqlmodel.PermissionOperation {
	out := make([]*gqlmodel.PermissionOperation, 0, len(defs))
	for _, def := range defs {
		out = append(out, &gqlmodel.PermissionOperation{
			Operation:   string(def.Operation),
			DisplayName: text(def.DisplayName),
			Description: text(def.Description),
			Implies:     operationStrings(def.Implies),
		})
	}
	return out
}

func resourceModel(
	text textFunc,
	def *permissionspec.ResourceDefinition,
	enforceable []permission.DataScope,
) *gqlmodel.PermissionResource {
	scopes := make([]string, len(enforceable))
	for i, scope := range enforceable {
		scopes[i] = string(scope)
	}

	var parent *string
	if def.ParentResource != "" {
		value := def.ParentResource
		parent = &value
	}

	return &gqlmodel.PermissionResource{
		Resource:           def.Resource,
		DisplayName:        text(def.DisplayName),
		Description:        text(def.Description),
		Category:           text(def.Category),
		Operations:         operationModels(text, def.Operations),
		ParentResource:     parent,
		DefaultSensitivity: string(def.DefaultSensitivity),
		EnforceableScopes:  scopes,
	}
}

func resourceCategories(
	ctx context.Context,
	registry *permissioncatalog.Registry,
	scopes permissionports.ScopePolicy,
) []*gqlmodel.PermissionResourceCategory {
	categories := registry.GetCategories()
	out := make([]*gqlmodel.PermissionResourceCategory, 0, len(categories))
	for _, category := range categories {
		defs := registry.GetByCategory(category)
		resources := make([]*gqlmodel.PermissionResource, 0, len(defs))
		for _, def := range defs {
			resources = append(
				resources,
				resourceModel(translator(ctx), def, scopes.EnforceableScopes(def.Resource)),
			)
		}
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].DisplayName < resources[j].DisplayName
		})
		out = append(out, &gqlmodel.PermissionResourceCategory{
			Category:  i18n.T(ctx, category),
			Resources: resources,
		})
	}
	return out
}

func allowedResourceCategories(
	categories []apikeyservice.AllowedResourceCategory,
) []*gqlmodel.PermissionResourceCategory {
	out := make([]*gqlmodel.PermissionResourceCategory, 0, len(categories))
	for _, category := range categories {
		resources := make([]*gqlmodel.PermissionResource, 0, len(category.Resources))
		for _, def := range category.Resources {
			resources = append(resources, resourceModel(verbatim, def, nil))
		}
		out = append(out, &gqlmodel.PermissionResourceCategory{
			Category:  category.Category,
			Resources: resources,
		})
	}
	return out
}

type bundleGrantSource struct {
	resource   string
	operations []permission.Operation
	scope      permission.DataScope
}

func bundleGrant(
	ctx context.Context,
	registry *permissioncatalog.Registry,
	scopes permissionports.ScopePolicy,
	grant bundleGrantSource,
) *gqlmodel.PermissionBundleGrant {
	scope := grant.scope
	if scope == "" || !scopes.Enforced(grant.resource, scope) {
		scope = permission.DataScopeOrganization
	}
	name := grant.resource
	if def, ok := registry.Get(grant.resource); ok {
		name = i18n.T(ctx, def.DisplayName)
	}

	return &gqlmodel.PermissionBundleGrant{
		Resource:    grant.resource,
		DisplayName: name,
		Operations:  operationStrings(grant.operations),
		DataScope:   string(scope),
	}
}

func bundleCatalog(
	ctx context.Context,
	registry *permissioncatalog.Registry,
	scopes permissionports.ScopePolicy,
) *gqlmodel.PermissionBundleCatalog {
	bundles := permissioncatalog.Bundles()
	templates := permissioncatalog.JobTemplates()

	catalog := &gqlmodel.PermissionBundleCatalog{
		Bundles:   make([]*gqlmodel.PermissionBundle, 0, len(bundles)),
		Templates: make([]*gqlmodel.PermissionJobTemplate, 0, len(templates)),
	}
	for _, bundle := range bundles {
		grants := make([]*gqlmodel.PermissionBundleGrant, 0, len(bundle.Grants))
		for _, grant := range bundle.Grants {
			grants = append(grants, bundleGrant(ctx, registry, scopes, bundleGrantSource{
				resource:   grant.Resource.String(),
				operations: grant.Operations,
				scope:      grant.Scope,
			}))
		}
		catalog.Bundles = append(catalog.Bundles, &gqlmodel.PermissionBundle{
			ID:          bundle.ID,
			Module:      bundle.Module,
			Label:       i18n.T(ctx, bundle.Label),
			Description: i18n.T(ctx, bundle.Description),
			Grants:      grants,
		})
	}
	for i := range templates {
		template := &templates[i]
		composed := registry.TemplateGrants(template)
		grants := make([]*gqlmodel.PermissionBundleGrant, 0, len(composed))
		for _, rp := range composed {
			grants = append(grants, bundleGrant(ctx, registry, scopes, bundleGrantSource{
				resource:   rp.Resource,
				operations: rp.Operations,
				scope:      rp.DataScope,
			}))
		}
		bundleIDs := template.Bundles
		if bundleIDs == nil {
			bundleIDs = []string{}
		}
		catalog.Templates = append(catalog.Templates, &gqlmodel.PermissionJobTemplate{
			ID:             template.ID,
			Label:          i18n.T(ctx, template.Label),
			Description:    i18n.T(ctx, template.Description),
			Bundles:        bundleIDs,
			MaxSensitivity: string(template.MaxSensitivity),
			Grants:         grants,
		})
	}

	return catalog
}

func idStrings(ids []pulid.ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

func roleSummaries(roles []permissionports.RoleSummary) []*gqlmodel.PermissionRoleSummary {
	out := make([]*gqlmodel.PermissionRoleSummary, 0, len(roles))
	for _, role := range roles {
		out = append(out, &gqlmodel.PermissionRoleSummary{
			ID:              role.ID.String(),
			Name:            role.Name,
			Description:     role.Description,
			IsSystem:        role.IsSystem,
			PermissionCount: role.PermissionCount,
		})
	}
	return out
}

func manifestToModel(manifest *permissionports.LightPermissionManifest) *gqlmodel.PermissionManifest {
	grants := make([]*gqlmodel.PermissionGrantMask, 0, len(manifest.Permissions))
	for resource, mask := range manifest.Permissions {
		grants = append(grants, &gqlmodel.PermissionGrantMask{
			Resource:   resource,
			Operations: int(mask),
		})
	}
	sort.Slice(grants, func(i, j int) bool { return grants[i].Resource < grants[j].Resource })

	routes := make([]*gqlmodel.PermissionRouteAccess, 0, len(manifest.RouteAccess))
	for route, allowed := range manifest.RouteAccess {
		routes = append(routes, &gqlmodel.PermissionRouteAccess{Route: route, Allowed: allowed})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Route < routes[j].Route })

	orgs := make([]*gqlmodel.PermissionOrganizationSummary, 0, len(manifest.AvailableOrgs))
	for _, org := range manifest.AvailableOrgs {
		orgs = append(orgs, &gqlmodel.PermissionOrganizationSummary{
			ID:   org.ID.String(),
			Name: org.Name,
		})
	}

	return &gqlmodel.PermissionManifest{
		Version:                manifest.Version,
		UserID:                 manifest.UserID.String(),
		OrganizationID:         manifest.OrganizationID.String(),
		ActiveRoleIds:          idStrings(manifest.ActiveRoleIDs),
		AuthorizedRoleIds:      idStrings(manifest.AuthorizedRoleIDs),
		ActiveRoles:            roleSummaries(manifest.ActiveRoles),
		AuthorizedRoles:        roleSummaries(manifest.AuthorizedRoles),
		RequiresRoleActivation: manifest.RequiresRoleActivation,
		MaxSensitivity:         string(manifest.MaxSensitivity),
		Permissions:            grants,
		RouteAccess:            routes,
		AvailableOrgs:          orgs,
		Checksum:               manifest.Checksum,
		ExpiresAt:              int(manifest.ExpiresAt),
	}
}
