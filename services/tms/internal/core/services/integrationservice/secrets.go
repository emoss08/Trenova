package integrationservice

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type secretScope struct {
	tenant pagination.TenantInfo
	typ    integration.Type
	bind   bool
}

func newSecretScope(
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	spec integration.IntegrationSpec,
) secretScope {
	return secretScope{tenant: tenantInfo, typ: typ, bind: spec.BindSecretsToTenant}
}

func (sc secretScope) aad(fieldKey string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIntegrationSecret,
		OrganizationID: sc.tenant.OrgID,
		BusinessUnitID: sc.tenant.BuID,
		ResourceID:     "integration:" + sc.typ.String() + ":" + fieldKey,
	}
}

func (s *Service) encryptSecret(value, fieldKey string, scope secretScope) (string, error) {
	if !scope.bind {
		return s.encryption.EncryptString(value)
	}
	return s.encryption.EncryptStringWithAAD(value, scope.aad(fieldKey))
}

func (s *Service) decryptSecret(value, fieldKey string, scope secretScope) (string, error) {
	if !scope.bind {
		return s.encryption.DecryptString(value)
	}
	decrypted, err := s.encryption.DecryptStringWithAAD(value, scope.aad(fieldKey))
	if err == nil {
		return decrypted, nil
	}
	legacy, legacyErr := s.encryption.DecryptString(value)
	if legacyErr != nil {
		return "", err
	}
	return legacy, nil
}

func (s *Service) rebindLegacySecret(stored, fieldKey string, scope secretScope) string {
	if !scope.bind || stored == "" {
		return stored
	}
	if _, err := s.encryption.DecryptStringWithAAD(stored, scope.aad(fieldKey)); err == nil {
		return stored
	}
	plaintext, err := s.encryption.DecryptString(stored)
	if err != nil {
		return stored
	}
	rebound, err := s.encryption.EncryptStringWithAAD(plaintext, scope.aad(fieldKey))
	if err != nil {
		s.l.Warn("failed to bind integration secret to tenant", zap.Error(err),
			zap.String("type", scope.typ.String()))
		return stored
	}
	return rebound
}
