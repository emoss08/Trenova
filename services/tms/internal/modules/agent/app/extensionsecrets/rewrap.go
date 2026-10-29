package extensionsecrets

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension/agentextensiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal/extensionportaldb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch/extensionwebwatchdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	extension := agentextensiondb.Extensions
	site := extensionportaldb.PortalSites
	watch := extensionwebwatchdb.WebWatches
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
				scope := ExtensionScope(row.Tenant, typ)
				return scope.SecretScopes(field)
			},
		},
		{
			Name:    "agent portal site credentials",
			Value:   site.Credentials,
			JSONMap: true,
			Rebind:  true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := PortalSiteScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
		{
			Name:    "agent web watch secrets",
			Value:   watch.Secrets,
			JSONMap: true,
			Rebind:  true,
			Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
				scope := WebWatchScope(row.Tenant, pulid.ID(row.ID))
				return scope.SecretScopes(field)
			},
		},
	}
}
