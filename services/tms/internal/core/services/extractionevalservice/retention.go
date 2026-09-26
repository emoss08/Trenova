package extractionevalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	purgeBatchSize  = 500
	purgeMaxBatches = 100
)

func (s *Service) PurgeExpiredRuns(
	ctx context.Context,
	req services.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	if s.retention == nil {
		return 0, nil
	}

	settings, err := s.retention.Get(ctx, repositories.GetDataRetentionRequest{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return 0, err
	}
	if err != nil {
		settings = &tenant.DataRetention{}
	}

	now := req.Now
	if now <= 0 {
		now = s.now()
	}
	before := now - int64(settings.AICorrectionRetentionDays())*timeutils.SecondsPerDay

	var total int64
	for range purgeMaxBatches {
		purged, pErr := s.runs.PurgeBefore(ctx, repositories.PurgeExtractionEvalRunsRequest{
			TenantInfo: req.TenantInfo,
			Before:     before,
			Limit:      purgeBatchSize,
		})
		if pErr != nil {
			return total, pErr
		}
		total += purged
		if purged < purgeBatchSize {
			break
		}
	}

	return total, nil
}
