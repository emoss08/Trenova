package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/aicorrectionretention"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const purgeBatchSize = 1000

func (s *Service) PurgeExpiredAssignments(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	sweep := aicorrectionretention.Sweep{
		Retention: s.retention,
		BatchSize: purgeBatchSize,
		Now:       s.now,
	}

	return sweep.Run(ctx, req, func(ctx context.Context, before int64, limit int) (int64, error) {
		return s.assignments.PurgeBefore(ctx, extractionrolloutports.PurgeRolloutAssignmentsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
