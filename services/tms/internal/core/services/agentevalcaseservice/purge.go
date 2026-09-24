package agentevalcaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/zap"
)

const (
	defaultPurgeBatch = 200
	maxPurgeBatches   = 500
	secondsPerDay     = int64(24 * 60 * 60)
)

func (s *Service) Purge(
	ctx context.Context,
	req services.PurgeEvalCasesRequest,
) (*services.PurgeEvalCasesResult, error) {
	now := req.Now
	if now <= 0 {
		now = s.now()
	}
	limit := intutils.WithDefault(req.Limit, defaultPurgeBatch)
	result := &services.PurgeEvalCasesResult{}

	retentions, err := s.retention.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, retention := range retentions.Items {
		if retention.AgentEvalCaseRetentionPeriod <= 0 {
			continue
		}
		purged, purgeErr := drain(limit, func() (int, error) {
			return s.cases.PurgeExpired(ctx, repositories.PurgeExpiredEvalCasesRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: retention.OrganizationID,
					BuID:  retention.BusinessUnitID,
				},
				CreatedBefore: now - int64(retention.AgentEvalCaseRetentionPeriod)*secondsPerDay,
				Now:           now,
				Limit:         limit,
			})
		})
		result.Retained += purged
		if purgeErr != nil {
			s.l.Error("evaluation case retention failed for an organization",
				zap.String("organizationId", retention.OrganizationID.String()),
				zap.Error(purgeErr),
			)
		}
	}

	result.Expired, err = drain(limit, func() (int, error) {
		return s.cases.PurgeExpired(ctx, repositories.PurgeExpiredEvalCasesRequest{
			AllTenants: true,
			Now:        now,
			Limit:      limit,
		})
	})
	if err != nil {
		return result, err
	}

	result.Orphaned, err = drain(limit, func() (int, error) {
		return s.cases.PurgeOrphaned(ctx, repositories.PurgeOrphanedEvalCasesRequest{Limit: limit})
	})
	if err != nil {
		return result, err
	}

	return result, nil
}

func drain(limit int, batch func() (int, error)) (int, error) {
	total := 0
	for range maxPurgeBatches {
		purged, err := batch()
		if err != nil {
			return total, err
		}
		total += purged
		if purged < limit {
			return total, nil
		}
	}

	return total, nil
}
