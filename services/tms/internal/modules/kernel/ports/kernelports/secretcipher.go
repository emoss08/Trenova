package kernelports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type SecretPurpose string

const SecretPurposeFactoringTransport SecretPurpose = "factoring_transport_secret" // #nosec G101 -- AAD label, not a credential.

type SecretScope struct {
	Purpose    SecretPurpose
	Tenant     pagination.TenantInfo
	ResourceID string
}

type SecretCipher interface {
	Seal(ctx context.Context, plaintext string, scope *SecretScope) (string, error)
	Open(ctx context.Context, sealed string, scope *SecretScope) (string, error)
}
