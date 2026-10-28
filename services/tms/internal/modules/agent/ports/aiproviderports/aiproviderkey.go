package aiproviderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
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

// RecordAdvisorRefusalRequest is the API refusing the advisor a provider
// declared for its model. Advisor and Model are what the call carried, so a
// refusal that raced a change to either is not recorded against the new one.
type RecordAdvisorRefusalRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Refusal    aiprovider.AdvisorRefusal
}

// AIProviderAdvisorRecorder keeps the API's refusal of a provider's advisor on
// the provider, where AI control shows it.
type AIProviderAdvisorRecorder interface {
	RecordAdvisorRefusal(ctx context.Context, req *RecordAdvisorRefusalRequest) error
}
