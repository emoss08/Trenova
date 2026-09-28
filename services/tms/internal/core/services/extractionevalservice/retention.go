package extractionevalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/aicorrectionretention"
)

const purgeBatchSize = 500

func (s *Service) PurgeExpiredRuns(
	ctx context.Context,
	req services.PurgeExpiredAICorrectionsRequest,
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
		return s.runs.PurgeBefore(ctx, repositories.PurgeExtractionEvalRunsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
