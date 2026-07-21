package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
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
) *dataloader.Loader[string, *costingservice.ShipmentProfitabilityEstimate] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentProfitabilityLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *costingservice.ShipmentProfitabilityEstimate] {
	return func(
		ctx context.Context,
		keys []string,
	) []*dataloader.Result[*costingservice.ShipmentProfitabilityEstimate] {
		results := make(
			[]*dataloader.Result[*costingservice.ShipmentProfitabilityEstimate],
			len(keys),
		)

		shipmentIDs := make([]pulid.ID, 0, len(keys))
		for _, key := range keys {
			parsed, err := pulid.MustParse(key)
			if err != nil {
				continue
			}
			shipmentIDs = append(shipmentIDs, parsed)
		}

		estimates, err := f.costingService.EstimateShipments(ctx, tenantInfo, shipmentIDs)
		if err != nil {
			for i := range results {
				results[i] = &dataloader.Result[*costingservice.ShipmentProfitabilityEstimate]{
					Error: err,
				}
			}
			return results
		}

		for i, key := range keys {
			parsed, parseErr := pulid.MustParse(key)
			if parseErr != nil {
				results[i] = &dataloader.Result[*costingservice.ShipmentProfitabilityEstimate]{
					Error: parseErr,
				}
				continue
			}
			results[i] = &dataloader.Result[*costingservice.ShipmentProfitabilityEstimate]{
				Data: estimates[parsed],
			}
		}

		return results
	}
}
