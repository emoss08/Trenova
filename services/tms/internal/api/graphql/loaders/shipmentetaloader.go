package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type ShipmentEtaLoaderFactoryParams struct {
	fx.In

	Etas services.ShipmentEtaReader
}

type ShipmentEtaLoaderFactory struct {
	etas services.ShipmentEtaReader
}

func NewShipmentEtaLoaderFactory(p ShipmentEtaLoaderFactoryParams) *ShipmentEtaLoaderFactory {
	return &ShipmentEtaLoaderFactory{etas: p.Etas}
}

func (f *ShipmentEtaLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *services.ShipmentEta] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentEtaLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*services.ShipmentEta] {
	return func(ctx context.Context, keys []string) ([]*services.ShipmentEta, []error) {
		values := make([]*services.ShipmentEta, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		etas, err := f.etas.EtasByShipmentIDs(ctx, tenantInfo, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = etas[id]
			}
		}

		return values, errs
	}
}
