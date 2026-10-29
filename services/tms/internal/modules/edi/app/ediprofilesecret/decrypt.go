package ediprofilesecret

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) decryptProfileSecret(
	profile *edi.EDICommunicationProfile,
	key string,
) (string, error) {
	if profile == nil || len(profile.EncryptedSecrets) == 0 {
		return "", nil
	}
	value := strings.TrimSpace(profile.EncryptedSecrets[key])
	if value == "" {
		return "", nil
	}
	if s.encryption == nil {
		if encryptionservice.IsEnvelope(value) {
			return "", errors.New(
				"encrypted EDI communication profile secret cannot be decrypted without encryption service",
			)
		}
		return value, nil
	}
	if !encryptionservice.IsEnvelope(value) {
		return value, nil
	}
	return s.encryption.DecryptStringWithAAD(value, ProfileSecretAAD(
		pagination.TenantInfo{OrgID: profile.OrganizationID, BuID: profile.BusinessUnitID},
		profile.ID,
		key,
	))
}
