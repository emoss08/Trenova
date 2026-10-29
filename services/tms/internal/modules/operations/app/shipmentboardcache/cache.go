package shipmentboardcache

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type Request struct {
	Section    shipmentboardports.ShipmentBoardSection
	TenantInfo pagination.TenantInfo
	Timezone   string
	Variant    string
}

func GetOrCompute[T any](
	ctx context.Context,
	cache shipmentboardports.ShipmentBoardCache,
	logger *zap.Logger,
	req *Request,
	compute func(ctx context.Context) (*T, error),
) (*T, error) {
	if cache == nil {
		return compute(ctx)
	}

	epoch, err := cache.Epoch(ctx, req.TenantInfo)
	if err != nil {
		logger.Warn("shipment board cache unavailable", zap.Error(err))
		return compute(ctx)
	}

	key := &shipmentboardports.ShipmentBoardCacheKey{
		Section:    req.Section,
		TenantInfo: req.TenantInfo,
		Timezone:   req.Timezone,
		Epoch:      epoch,
		Variant:    req.Variant,
	}

	cached := new(T)
	hit, err := cache.Get(ctx, key, cached)
	if err != nil {
		logger.Warn("shipment board cache read failed", zap.Error(err))
	}
	if hit {
		return cached, nil
	}

	value, err := compute(ctx)
	if err != nil {
		return nil, err
	}

	if err = cache.Set(ctx, key, value); err != nil {
		logger.Warn("shipment board cache write failed", zap.Error(err))
	}

	return value, nil
}
