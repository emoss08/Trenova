package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type ShipmentProfitabilityLoaderFactoryParams struct {
	fx.In

	CostingService *costingservice.Service
}

type ShipmentProfitabilityLoaderFactory struct {
	costingService *costingservice.Service
}

func NewShipmentProfitabilityLoaderFactory(
	p ShipmentProfitabilityLoaderFactoryParams,
) *ShipmentProfitabilityLoaderFactory {
	return &ShipmentProfitabilityLoaderFactory{
		costingService: p.CostingService,
	}
}

func (f *ShipmentProfitabilityLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentProfitabilityLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*costingservice.ShipmentProfitabilityEstimate] {
	return func(
		ctx context.Context,
		keys []string,
	) ([]*costingservice.ShipmentProfitabilityEstimate, []error) {
		values := make([]*costingservice.ShipmentProfitabilityEstimate, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		estimates, err := f.costingService.EstimateShipments(ctx, tenantInfo, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = estimates[id]
			}
		}

		return values, errs
	}
}
