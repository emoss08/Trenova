package aicorrectionretention

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxBatches = 100

type Reader interface {
	Get(
		ctx context.Context,
		req tenantports.GetDataRetentionRequest,
	) (*dataretention.DataRetention, error)
}

type PurgeFunc func(ctx context.Context, before int64, limit int) (int64, error)

type Sweep struct {
	Retention Reader
	BatchSize int
	Now       func() int64
}

func (s Sweep) Run(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
	purge PurgeFunc,
) (int64, error) {
	before, err := s.cutoff(ctx, req)
	if err != nil {
		return 0, err
	}

	var total int64
	for range maxBatches {
		purged, pErr := purge(ctx, before, s.BatchSize)
		if pErr != nil {
			return total, pErr
		}
		total += purged
		if purged < int64(s.BatchSize) {
			break
		}
	}

	return total, nil
}

func (s Sweep) cutoff(
	ctx context.Context,
	req kernelports.PurgeExpiredAICorrectionsRequest,
) (int64, error) {
	settings, err := s.Retention.Get(ctx, tenantports.GetDataRetentionRequest{
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	})
	switch {
	case err == nil:
	case errortypes.IsNotFoundError(err):
		settings = &dataretention.DataRetention{}
	default:
		return 0, err
	}

	now := req.Now
	if now <= 0 {
		now = s.Now()
	}

	return now - int64(settings.AICorrectionRetentionDays())*timeutils.SecondsPerDay, nil
}
