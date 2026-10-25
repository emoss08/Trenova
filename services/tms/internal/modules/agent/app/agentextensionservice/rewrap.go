package agentextensionservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	extension := agentextensiondb.Extensions
	site := agentextensiondb.PortalSites
	watch := agentextensiondb.WebWatches
	extensionType := kernelports.EnvelopeColumnName(extension.Type)

	return []kernelports.EnvelopeColumn{
		{
			Name:    "agent extension secrets",
			Value:   extension.Configuration,
			Extra:   []keel.Expr{extension.Type},
			JSONMap: true,
			Rebind:  true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				typ := agentextension.Type(row.Extra[extensionType])
				scope := secretScope(row.Tenant, typ)
				return scope.SecretScopes(field)
			},
		},
		{
			Name:    "agent portal site credentials",
			Value:   site.Credentials,
			JSONMap: true,
			Rebind:  true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := portalSiteScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
		{
			Name:    "agent web watch secrets",
			Value:   watch.Secrets,
			JSONMap: true,
			Rebind:  true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := webWatchScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
	}
}
