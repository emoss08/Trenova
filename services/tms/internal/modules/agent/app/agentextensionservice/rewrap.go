package agentextensionservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/agentextensioncol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	extension := agentextensioncol.ExtensionColumns
	site := agentextensioncol.PortalSiteColumns
	watch := agentextensioncol.WebWatchColumns

	return []kernelports.EnvelopeColumn{
		{
			Name:           "agent extension secrets",
			Table:          agentextensioncol.ExtensionTable,
			ID:             extension.ID,
			OrganizationID: extension.OrganizationID,
			BusinessUnitID: extension.BusinessUnitID,
			Value:          extension.Configuration,
			Extra:          []buncolgen.Column{extension.Type},
			JSONMap:        true,
			Rebind:         true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				typ := agentextension.Type(row.Extra[extension.Type.Name])
				scope := secretScope(row.Tenant, typ)
				return scope.SecretScopes(field)
			},
		},
		{
			Name:           "agent portal site credentials",
			Table:          agentextensioncol.PortalSiteTable,
			ID:             site.ID,
			OrganizationID: site.OrganizationID,
			BusinessUnitID: site.BusinessUnitID,
			Value:          site.Credentials,
			JSONMap:        true,
			Rebind:         true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := portalSiteScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
		{
			Name:           "agent web watch secrets",
			Table:          agentextensioncol.WebWatchTable,
			ID:             watch.ID,
			OrganizationID: watch.OrganizationID,
			BusinessUnitID: watch.BusinessUnitID,
			Value:          watch.Secrets,
			JSONMap:        true,
			Rebind:         true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := webWatchScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
	}
}
