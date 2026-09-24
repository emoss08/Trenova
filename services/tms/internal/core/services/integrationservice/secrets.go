package integrationservice

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/secretconfig"
	"github.com/emoss08/trenova/pkg/pagination"
)

func newSecretScope(
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	spec integration.IntegrationSpec,
) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeIntegrationSecret,
		Tenant:       tenantInfo,
		ResourceKind: "integration",
		Subject:      typ.String(),
		Bind:         spec.BindSecretsToTenant,
	}
}
