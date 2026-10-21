package encryptionservice

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
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

var _ kernelports.EnvelopeRewrapper = (*SecretCipher)(nil)

func NewEnvelopeRewrapper(service *Service) kernelports.EnvelopeRewrapper {
	return &SecretCipher{service: service}
}

func (c *SecretCipher) IsEnvelope(value string) bool {
	return IsEnvelope(value)
}

func (c *SecretCipher) Rewrap(
	ctx context.Context,
	req *kernelports.EnvelopeRewrapRequest,
) (rewrapped string, changed bool, err error) {
	if len(req.Scopes) == 0 {
		return "", false, ErrInvalidEnvelope
	}

	target := aadFor(&req.Scopes[0])
	for i := range req.Scopes {
		aad := aadFor(&req.Scopes[i])
		if !EnvelopeHasAAD(req.Sealed, aad) {
			continue
		}
		if i == 0 || !req.Rebind {
			return c.service.RewrapEnvelope(ctx, req.Sealed, aad)
		}
		plaintext, decryptErr := c.service.DecryptBytesWithAADContext(ctx, req.Sealed, aad)
		if decryptErr != nil {
			return "", false, decryptErr
		}
		rewrapped, err = c.service.EncryptBytesWithAADContext(ctx, plaintext, target)
		if err != nil {
			return "", false, err
		}
		return rewrapped, true, nil
	}
	return "", false, ErrInvalidEnvelope
}

func (c *SecretCipher) IsBlob(prefix []byte) bool {
	return IsBlob(prefix)
}

func (c *SecretCipher) BlobPrefixSize() int {
	return BlobPrefixSize()
}

func (c *SecretCipher) RewrapBlob(
	ctx context.Context,
	sealed io.Reader,
	scopes []kernelports.SecretScope,
) (*kernelports.BlobRewrap, error) {
	aads := make([]AAD, 0, len(scopes))
	for i := range scopes {
		aads = append(aads, aadFor(&scopes[i]))
	}
	rewrapped, changed, err := c.service.RewrapBlob(ctx, sealed, aads)
	if err != nil {
		return nil, err
	}
	return &kernelports.BlobRewrap{
		Header:   rewrapped.Header,
		Consumed: rewrapped.Consumed,
		Changed:  changed,
	}, nil
}

func (a AAD) SecretScope() kernelports.SecretScope {
	return kernelports.SecretScope{
		Purpose:    kernelports.SecretPurpose(a.Purpose),
		Tenant:     pagination.TenantInfo{OrgID: a.OrganizationID, BuID: a.BusinessUnitID},
		ResourceID: a.ResourceID,
	}
}
