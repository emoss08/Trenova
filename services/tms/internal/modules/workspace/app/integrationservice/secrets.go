package integrationservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
)

func newSecretScope(tenantInfo pagination.TenantInfo, typ integration.Type) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeIntegrationSecret,
		Tenant:       tenantInfo,
		ResourceKind: "integration",
		Subject:      typ.String(),
	}
}

func (s *Service) OpenSecret(
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	fieldKey, sealed string,
) (string, error) {
	return s.secrets.Decrypt(sealed, fieldKey, newSecretScope(tenantInfo, typ))
}
