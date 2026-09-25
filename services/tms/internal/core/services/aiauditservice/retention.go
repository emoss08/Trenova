package aiauditservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	pruneBatchSize = 5000
	secondsPerDay  = 24 * 60 * 60
)

// PruneResult is what one retention sweep removed.
type PruneResult struct {
	Deleted int `json:"deleted"`
	Tenants int `json:"tenants"`
	Failed  int `json:"failed"`
}

// Retention removes the oldest part of each tenant's trail once it is past
// the organization's AI audit retention period. It cuts only at the end of a
// seal, so the oldest row left still links to the seal before it, and it
// keeps every seal.
type Retention struct {
	ledger    repositories.AIAuditRepository
	retention repositories.DataRetentionRepository
	metrics   *metrics.AIAudit
	now       func() time.Time
	l         *zap.Logger
}

func NewRetention(
	ledger repositories.AIAuditRepository,
	retention repositories.DataRetentionRepository,
	registry *metrics.AIAudit,
	logger *zap.Logger,
) *Retention {
	return &Retention{
		ledger:    ledger,
		retention: retention,
		metrics:   registry,
		now:       time.Now,
		l:         logger.Named("aiaudit.retention"),
	}
}

// PruneAll sweeps every tenant with a trail. A tenant with no retention
// settings keeps the default seven years.
func (r *Retention) PruneAll(ctx context.Context, heartbeat Heartbeat) (*PruneResult, error) {
	heads, err := r.ledger.ListChainHeads(ctx)
	if err != nil {
		return nil, err
	}
	if len(heads) == 0 {
		return &PruneResult{}, nil
	}

	settings, err := r.retention.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("read AI audit retention settings: %w", err)
	}
	periods := make(map[pagination.TenantInfo]int, len(settings.Items))
	for _, item := range settings.Items {
		periods[pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID}] =
			item.AIAuditRetentionDays()
	}

	result := &PruneResult{}
	var failures []error
	now := r.now().Unix()
	for _, head := range heads {
		tenantInfo := pagination.TenantInfo{OrgID: head.OrganizationID, BuID: head.BusinessUnitID}
		days, ok := periods[tenantInfo]
		if !ok || days <= 0 {
			days = tenant.DefaultAIAuditRetentionDays
		}
		days = max(days, tenant.MinAIAuditRetentionDays)

		deleted, pruneErr := r.pruneTenant(ctx, tenantInfo, now-int64(days)*secondsPerDay)
		if pruneErr != nil {
			failures = append(failures, pruneErr)
			result.Failed++

			continue
		}
		if deleted > 0 {
			result.Tenants++
			result.Deleted += deleted
		}
		beat(heartbeat, tenantInfo.OrgID.String(), deleted)
	}
	r.metrics.RecordPruned(result.Deleted)

	if len(failures) > 0 && result.Failed == len(heads) {
		return result, errors.Join(failures...)
	}
	for _, failure := range failures {
		r.l.Error("failed to prune one tenant's AI audit trail", zap.Error(failure))
	}

	return result, nil
}

func (r *Retention) pruneTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cutoff int64,
) (int, error) {
	seal, err := r.ledger.LastSealBefore(ctx, tenantInfo, cutoff)
	if err != nil || seal == nil {
		return 0, err
	}

	deleted, err := r.ledger.Prune(ctx, repositories.PruneAIAuditEventsRequest{
		TenantInfo: tenantInfo,
		ThroughSeq: seal.ToSeq,
		BatchSize:  pruneBatchSize,
	})
	if err != nil {
		return deleted, fmt.Errorf("prune the AI audit trail of %s: %w", tenantInfo.OrgID, err)
	}

	return deleted, nil
}
