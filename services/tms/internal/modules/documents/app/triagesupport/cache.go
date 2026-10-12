package triagesupport

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type Request struct {
	Section    triageports.CacheSection
	TenantInfo pagination.TenantInfo
	Variant    string
	TTL        time.Duration
}

func GetOrCompute[T any](
	ctx context.Context,
	cache triageports.TriageCache,
	logger *zap.Logger,
	req *Request,
	compute func(ctx context.Context) (*T, error),
) (*T, error) {
	if cache == nil {
		return compute(ctx)
	}

	epoch, err := cache.QueueEpoch(ctx, req.TenantInfo)
	if err != nil {
		logger.Warn("triage cache unavailable", zap.String("section", string(req.Section)),
			zap.Error(err))

		return compute(ctx)
	}

	key := &triageports.CacheKey{
		Section:    req.Section,
		TenantInfo: req.TenantInfo,
		Epoch:      epoch,
		Variant:    req.Variant,
	}

	cached := new(T)
	hit, err := cache.Get(ctx, key, cached)
	if err != nil {
		logger.Warn("triage cache read failed", zap.String("section", string(req.Section)),
			zap.Error(err))
	}
	if hit {
		return cached, nil
	}

	value, err := compute(ctx)
	if err != nil {
		return nil, err
	}

	if err = cache.Set(ctx, key, value, req.TTL); err != nil {
		logger.Warn("triage cache write failed", zap.String("section", string(req.Section)),
			zap.Error(err))
	}

	return value, nil
}

func Invalidate(
	ctx context.Context,
	cache triageports.TriageCache,
	logger *zap.Logger,
	tenantInfo pagination.TenantInfo,
) {
	if cache == nil {
		return
	}
	if err := cache.BumpQueueEpoch(ctx, tenantInfo); err != nil {
		logger.Warn("failed to drop the triage caches",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))
	}
}
