package extractionevalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/aicorrectionretention"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const purgeBatchSize = 500

func (s *Service) PurgeExpiredRuns(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	if s.retention == nil {
		return 0, nil
	}

	sweep := aicorrectionretention.Sweep{
		Retention: s.retention,
		BatchSize: purgeBatchSize,
		Now:       s.now,
	}

	return sweep.Run(ctx, req, func(ctx context.Context, before int64, limit int) (int64, error) {
		return s.runs.PurgeBefore(ctx, extractionevalports.PurgeExtractionEvalRunsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
