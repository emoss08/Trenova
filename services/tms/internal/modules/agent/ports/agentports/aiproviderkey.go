package agentports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordAIProviderKeyFingerprintRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Prefix     string
	LastFour   string
}

type TouchAIProviderKeyRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	UsedAt     int64
}

type ClearExpiredAIProviderKeysRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

type RebindAIProviderKeyRequest struct {
	ID          pulid.ID
	TenantInfo  pagination.TenantInfo
	Previous    bool
	Expected    string
	Replacement string
}

type AIProviderKeyRepository interface {
	RebindSealedKey(ctx context.Context, req *RebindAIProviderKeyRequest) error
	RecordFingerprint(ctx context.Context, req RecordAIProviderKeyFingerprintRequest) error
	TouchUsed(ctx context.Context, req TouchAIProviderKeyRequest) error
	ClearExpiredPrevious(ctx context.Context, req ClearExpiredAIProviderKeysRequest) (int, error)
}
