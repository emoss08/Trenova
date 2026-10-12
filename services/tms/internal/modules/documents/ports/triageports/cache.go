package triageports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
)

type CacheSection string

const (
	CacheBriefing   = CacheSection("briefing")
	CachePreview    = CacheSection("preview")
	CacheSuggestion = CacheSection("suggestion")
)

type CacheKey struct {
	Section    CacheSection
	TenantInfo pagination.TenantInfo
	Epoch      int64
	Variant    string
}

type TriageCache interface {
	QueueEpoch(ctx context.Context, tenantInfo pagination.TenantInfo) (int64, error)
	BumpQueueEpoch(ctx context.Context, tenantInfo pagination.TenantInfo) error
	Get(ctx context.Context, key *CacheKey, dest any) (bool, error)
	Set(ctx context.Context, key *CacheKey, value any, ttl time.Duration) error
	EnabledRules(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*triage.Rule, bool, error)
	SetEnabledRules(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rules []*triage.Rule,
	) error
	DropEnabledRules(ctx context.Context, tenantInfo pagination.TenantInfo) error
	SenderHints(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		senderDomain string,
	) ([]*triage.SenderHint, bool, error)
	SetSenderHints(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		senderDomain string,
		hints []*triage.SenderHint,
	) error
	DropSenderHints(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		senderDomain string,
	) error
}
