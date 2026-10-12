package agentevalcaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/pkg/dbscope"
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
	req agentports.PurgeEvalCasesRequest,
) (*agentports.PurgeEvalCasesResult, error) {
	now := req.Now
	if now <= 0 {
		now = s.now()
	}
	limit := intutils.WithDefault(req.Limit, defaultPurgeBatch)
	result := &agentports.PurgeEvalCasesResult{}

	retentions, err := s.retention.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, retention := range retentions.Items {
		if retention.AgentEvalCaseRetentionPeriod <= 0 {
			continue
		}
		tenant := pagination.TenantInfo{
			OrgID: retention.OrganizationID,
			BuID:  retention.BusinessUnitID,
		}
		tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
		purged, purgeErr := drain(limit, func() (int, error) {
			return s.cases.PurgeExpired(tenantCtx, agentqualityports.PurgeExpiredEvalCasesRequest{
				TenantInfo:    tenant,
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
		return s.cases.PurgeExpired(ctx, agentqualityports.PurgeExpiredEvalCasesRequest{
			AllTenants: true,
			Now:        now,
			Limit:      limit,
		})
	})
	if err != nil {
		return result, err
	}

	result.Orphaned, err = drain(limit, func() (int, error) {
		return s.cases.PurgeOrphaned(
			ctx,
			agentqualityports.PurgeOrphanedEvalCasesRequest{Limit: limit},
		)
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
