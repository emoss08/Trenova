package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
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

func newShipmentProfitabilityLoader(f *ShipmentProfitabilityLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentProfitabilityLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*costingservice.ShipmentProfitabilityEstimate] {
	return func(
		ctx context.Context,
		keys []string,
	) ([]*costingservice.ShipmentProfitabilityEstimate, []error) {
		values := make([]*costingservice.ShipmentProfitabilityEstimate, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		estimates, err := f.costingService.EstimateShipments(ctx, tenantInfo, ids)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
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

var shipmentProfitabilityByIDKey = loadercore.NewKey(newShipmentProfitabilityLoader)

func ShipmentProfitabilityByID(
	ctx context.Context,
	key string,
) (*costingservice.ShipmentProfitabilityEstimate, error) {
	return loadercore.Load(ctx, shipmentProfitabilityByIDKey, key)
}

func (f *ShipmentProfitabilityLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
