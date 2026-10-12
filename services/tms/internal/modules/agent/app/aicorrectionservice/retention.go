package aicorrectionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/aicorrectionretention"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const purgeBatchSize = 1000

func (s *Service) PurgeExpired(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	sweep := aicorrectionretention.Sweep{
		Retention: s.retention,
		BatchSize: purgeBatchSize,
		Now:       s.now,
	}

	return sweep.Run(ctx, req, func(ctx context.Context, before int64, limit int) (int64, error) {
		return s.repo.PurgeBefore(ctx, aicorrectionports.PurgeAICorrectionsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      limit,
		})
	})
}
