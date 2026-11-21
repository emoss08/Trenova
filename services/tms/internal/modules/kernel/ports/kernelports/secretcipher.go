package kernelports

import (
	"context"
	"io"

	"github.com/emoss08/trenova/pkg/pagination"
)

type SecretPurpose string

const (
	SecretPurposeFactoringTransport SecretPurpose = "factoring_transport_secret" // #nosec G101 -- AAD label, not a credential.
	SecretPurposeSystemEmail        SecretPurpose = "system_email_credential"    // #nosec G101 -- AAD label, not a credential.
	SecretPurposeArchiveAccess      SecretPurpose = "tenant_archive_access"      // #nosec G101 -- AAD label, not a credential.
)

const (
	CryptoModeMetadataKey = "crypto_mode"
	CryptoModeBlobV1      = "blob_v1"
)

type SecretScope struct {
	Purpose    SecretPurpose
	Tenant     pagination.TenantInfo
	ResourceID string
}

type SecretCipher interface {
	Seal(ctx context.Context, plaintext string, scope *SecretScope) (string, error)
	Open(ctx context.Context, sealed string, scope *SecretScope) (string, error)
}

type OpenedBlob struct {
	Body  io.Reader
	Size  int64
	Scope SecretScope
}

type SealedBlob struct {
	Body io.Reader
	Size int64
}

type BlobCipher interface {
	IsBlob(prefix []byte) bool
	BlobPrefixSize() int
	OpenBlob(
		ctx context.Context,
		sealed io.Reader,
		sealedSize int64,
		scopes []SecretScope,
	) (*OpenedBlob, error)
	SealBlob(
		ctx context.Context,
		plain io.Reader,
		size int64,
		scope *SecretScope,
	) (*SealedBlob, error)
}
