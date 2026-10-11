package extractionshadowservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/aicorrectionretention"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const purgeBatchSize = 1000

func (s *Service) PurgeExpiredShadows(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	sweep := aicorrectionretention.Sweep{
		Retention: s.retention,
		BatchSize: purgeBatchSize,
		Now:       s.now,
	}

	return sweep.Run(ctx, req, func(ctx context.Context, before int64, limit int) (int64, error) {
		return s.results.PurgeBefore(ctx, extractionshadowports.PurgeExtractionShadowResultsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
