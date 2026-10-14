package encryptionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type SecretCipher struct {
	service *Service
}

var _ kernelports.SecretCipher = (*SecretCipher)(nil)

func NewSecretCipher(service *Service) kernelports.SecretCipher {
	return &SecretCipher{service: service}
}

func (c *SecretCipher) Seal(
	ctx context.Context,
	plaintext string,
	scope *kernelports.SecretScope,
) (string, error) {
	return c.service.EncryptBytesWithAADContext(ctx, []byte(plaintext), aadFor(scope))
}

func (c *SecretCipher) Open(
	ctx context.Context,
	sealed string,
	scope *kernelports.SecretScope,
) (string, error) {
	plaintext, err := c.service.DecryptBytesWithAADContext(ctx, sealed, aadFor(scope))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func aadFor(scope *kernelports.SecretScope) AAD {
	return AAD{
		Purpose:        Purpose(scope.Purpose),
		OrganizationID: scope.Tenant.OrgID,
		BusinessUnitID: scope.Tenant.BuID,
		ResourceID:     scope.ResourceID,
	}
}
