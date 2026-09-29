package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aicorrectionretention"
)

const purgeBatchSize = 1000

func (s *Service) PurgeExpiredAssignments(
	ctx context.Context,
	req services.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	sweep := aicorrectionretention.Sweep{
		Retention: s.retention,
		BatchSize: purgeBatchSize,
		Now:       s.now,
	}

	return sweep.Run(ctx, req, func(ctx context.Context, before int64, limit int) (int64, error) {
		return s.assignments.PurgeBefore(ctx, repositories.PurgeRolloutAssignmentsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
