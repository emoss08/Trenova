package extensionsecrets

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	secretResourceKind   = "agent_extension"
	portalSiteSecretKind = "agent_portal_site"
	webWatchSecretKind   = "agent_web_watch"
)

func PortalSiteScope(tenantInfo pagination.TenantInfo, siteID pulid.ID) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeAgentExtensionSecret,
		Tenant:       tenantInfo,
		ResourceKind: portalSiteSecretKind,
		Subject:      siteID.String(),
	}
}

func ExtensionScope(tenantInfo pagination.TenantInfo, typ agentextension.Type) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeAgentExtensionSecret,
		Tenant:       tenantInfo,
		ResourceKind: secretResourceKind,
		Subject:      typ.String(),
	}
}

func WebWatchScope(tenantInfo pagination.TenantInfo, watchID pulid.ID) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeAgentExtensionSecret,
		Tenant:       tenantInfo,
		ResourceKind: webWatchSecretKind,
		Subject:      watchID.String(),
	}
}
